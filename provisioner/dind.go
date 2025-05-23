package provisioner

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	lru "github.com/hashicorp/golang-lru"
	"github.com/ringo380/lessoncraft/config"
	"github.com/ringo380/lessoncraft/docker"
	"github.com/ringo380/lessoncraft/id"
	"github.com/ringo380/lessoncraft/lesson"
	"github.com/ringo380/lessoncraft/pwd/types"
	"github.com/ringo380/lessoncraft/router"
	"github.com/ringo380/lessoncraft/storage"
)

type DinD struct {
	factory   docker.FactoryApi
	storage   storage.StorageApi
	generator id.Generator
	cache     *lru.Cache
}

func NewDinD(generator id.Generator, f docker.FactoryApi, s storage.StorageApi) *DinD {
	c, _ := lru.New(5000)
	return &DinD{generator: generator, factory: f, storage: s, cache: c}
}

func checkHostnameExists(sessionId, hostname string, instances []*types.Instance) bool {
	exists := false
	for _, instance := range instances {
		if instance.Hostname == hostname {
			exists = true
			break
		}
	}
	return exists
}

func (d *DinD) InstanceNew(session *types.Session, conf types.InstanceConfig) (*types.Instance, error) {
	// Check if a lesson context is provided and if it specifies a custom Docker image or containers
	if conf.LessonCtx != nil && conf.LessonCtx.LessonID != "" {
		// Get the lesson from storage
		lessonData, err := d.storage.LessonGet(conf.LessonCtx.LessonID)
		if err == nil && lessonData != nil {
			// Check if the current step specifies containers or a custom Docker image
			if conf.LessonCtx.StepIndex >= 0 && conf.LessonCtx.StepIndex < len(lessonData.Steps) {
				currentStep := lessonData.Steps[conf.LessonCtx.StepIndex]

				// Check if the step has multiple containers defined
				if len(currentStep.Containers) > 0 {
					// For now, we only support creating the primary container
					// Multi-container support will be implemented in a future update

					// Find the primary container (role="primary" or first container if no primary is specified)
					var primaryContainer *lesson.ContainerConfig
					for i := range currentStep.Containers {
						if currentStep.Containers[i].Role == "primary" {
							primaryContainer = &currentStep.Containers[i]
							break
						}
					}

					// If no primary container was found, use the first container
					if primaryContainer == nil && len(currentStep.Containers) > 0 {
						primaryContainer = &currentStep.Containers[0]
					}

					// Use the primary container's image if found
					if primaryContainer != nil {
						conf.ImageName = primaryContainer.Image

						// Apply container-specific resource limits if specified
						if primaryContainer.MaxProcesses > 0 {
							conf.MaxProcesses = primaryContainer.MaxProcesses
						}
						if primaryContainer.MaxMemoryMB > 0 {
							conf.MaxMemoryMB = primaryContainer.MaxMemoryMB
						}
						if primaryContainer.StorageSize != "" {
							conf.StorageSize = primaryContainer.StorageSize
						}

						// Apply container-specific hostname if specified
						if primaryContainer.Hostname != "" {
							conf.Hostname = primaryContainer.Hostname
						}

						// Apply container-specific environment variables if specified
						if len(primaryContainer.Envs) > 0 {
							conf.Envs = append(conf.Envs, primaryContainer.Envs...)
						}

						// Apply container-specific networks if specified
						if len(primaryContainer.Networks) > 0 {
							conf.Networks = append(conf.Networks, primaryContainer.Networks...)
						}

						log.Printf("NewInstance - using container-specific image: [%s]\n", conf.ImageName)
					}
				} else if currentStep.Image != "" {
					// Fall back to the step's Image field for backward compatibility
					conf.ImageName = currentStep.Image
					log.Printf("NewInstance - using step-specific image: [%s]\n", conf.ImageName)
				} else if lessonData.DefaultImage != "" {
					// Use the lesson's default image if the step doesn't specify one
					conf.ImageName = lessonData.DefaultImage
					log.Printf("NewInstance - using lesson default image: [%s]\n", conf.ImageName)
				}

				// Apply step-specific resource limits if not already set by a container config
				if conf.MaxProcesses == 0 && currentStep.MaxProcesses > 0 {
					conf.MaxProcesses = currentStep.MaxProcesses
				}
				if conf.MaxMemoryMB == 0 && currentStep.MaxMemoryMB > 0 {
					conf.MaxMemoryMB = currentStep.MaxMemoryMB
				}
				if conf.StorageSize == "" && currentStep.StorageSize != "" {
					conf.StorageSize = currentStep.StorageSize
				}
			} else if lessonData.DefaultImage != "" {
				// Use the lesson's default image if no step is specified
				conf.ImageName = lessonData.DefaultImage
				log.Printf("NewInstance - using lesson default image: [%s]\n", conf.ImageName)
			}

			// Apply lesson-level default resource limits if not already set
			if conf.MaxProcesses == 0 && lessonData.DefaultMaxProcesses > 0 {
				conf.MaxProcesses = lessonData.DefaultMaxProcesses
			}
			if conf.MaxMemoryMB == 0 && lessonData.DefaultMaxMemoryMB > 0 {
				conf.MaxMemoryMB = lessonData.DefaultMaxMemoryMB
			}
			if conf.StorageSize == "" && lessonData.DefaultStorageSize != "" {
				conf.StorageSize = lessonData.DefaultStorageSize
			}
		}
	}

	// Fall back to playground default if no image is specified
	if conf.ImageName == "" {
		playground, err := d.storage.PlaygroundGet(session.PlaygroundId)
		if err != nil {
			return nil, err
		}
		conf.ImageName = playground.DefaultDinDInstanceImage
		log.Printf("NewInstance - using playground default image: [%s]\n", conf.ImageName)
	} else {
		log.Printf("NewInstance - using image: [%s]\n", conf.ImageName)
	}
	if conf.Hostname == "" {
		instances, err := d.storage.InstanceFindBySessionId(session.Id)
		if err != nil {
			return nil, err
		}
		var nodeName string
		for i := 1; ; i++ {
			nodeName = fmt.Sprintf("node%d", i)
			exists := checkHostnameExists(session.Id, nodeName, instances)
			if !exists {
				break
			}
		}
		conf.Hostname = nodeName
	}

	networks := []string{session.Id}
	if config.Unsafe {
		networks = append(networks, conf.Networks...)
	}

	containerName := fmt.Sprintf("%s_%s", session.Id[:8], d.generator.NewId())
	opts := docker.CreateContainerOpts{
		Image:          conf.ImageName,
		SessionId:      session.Id,
		ContainerName:  containerName,
		Hostname:       conf.Hostname,
		ServerCert:     conf.ServerCert,
		ServerKey:      conf.ServerKey,
		CACert:         conf.CACert,
		HostFQDN:       conf.PlaygroundFQDN,
		Privileged:     conf.Privileged,
		Networks:       networks,
		DindVolumeSize: conf.DindVolumeSize,
		Envs:           conf.Envs,
	}

	dockerClient, err := d.factory.GetForSession(session)
	if err != nil {
		return nil, err
	}
	if err := dockerClient.ContainerCreate(opts); err != nil {
		return nil, err
	}

	ips, err := dockerClient.ContainerIPs(containerName)
	if err != nil {
		return nil, err
	}

	instance := &types.Instance{}
	instance.Image = opts.Image
	instance.IP = ips[session.Id]
	instance.RoutableIP = instance.IP
	instance.SessionId = session.Id
	instance.Name = containerName
	instance.Hostname = conf.Hostname
	instance.Cert = conf.Cert
	instance.Key = conf.Key
	instance.ServerCert = conf.ServerCert
	instance.ServerKey = conf.ServerKey
	instance.CACert = conf.CACert
	instance.Tls = conf.Tls
	instance.ProxyHost = router.EncodeHost(session.Id, instance.RoutableIP, router.HostOpts{})
	instance.SessionHost = session.Host

	return instance, nil
}

func (d *DinD) getSession(sessionId string) (*types.Session, error) {
	var session *types.Session
	if s, found := d.cache.Get(sessionId); !found {
		s, err := d.storage.SessionGet(sessionId)
		if err != nil {
			return nil, err
		}
		session = s
		d.cache.Add(sessionId, s)
	} else {
		session = s.(*types.Session)
	}
	return session, nil
}

func (d *DinD) InstanceDelete(session *types.Session, instance *types.Instance) error {
	dockerClient, err := d.factory.GetForSession(session)
	if err != nil {
		return err
	}
	err = dockerClient.ContainerDelete(instance.Name)
	if err != nil && !strings.Contains(err.Error(), "No such container") {
		return err
	}
	return nil
}

func (d *DinD) InstanceExec(instance *types.Instance, cmd []string) (int, error) {
	session, err := d.getSession(instance.SessionId)
	if err != nil {
		return -1, err
	}
	dockerClient, err := d.factory.GetForSession(session)
	if err != nil {
		return -1, err
	}
	return dockerClient.Exec(instance.Name, cmd)
}

func (d *DinD) InstanceFSTree(instance *types.Instance) (io.Reader, error) {
	session, err := d.getSession(instance.SessionId)
	if err != nil {
		return nil, err
	}
	dockerClient, err := d.factory.GetForSession(session)
	if err != nil {
		return nil, err
	}
	b := bytes.NewBuffer([]byte{})

	if c, err := dockerClient.ExecAttach(instance.Name, []string{"bash", "-c", `tree --noreport -J $HOME`}, b); c > 0 {
		log.Println(b.String())
		return nil, fmt.Errorf("Error %d trying list directories", c)
	} else if err != nil {
		return nil, err
	}

	return b, nil
}

func (d *DinD) InstanceFile(instance *types.Instance, filePath string) (io.Reader, error) {
	session, err := d.getSession(instance.SessionId)
	if err != nil {
		return nil, err
	}
	dockerClient, err := d.factory.GetForSession(session)
	if err != nil {
		return nil, err
	}

	return dockerClient.CopyFromContainer(instance.Name, filePath)
}

func (d *DinD) InstanceResizeTerminal(instance *types.Instance, rows, cols uint) error {
	session, err := d.getSession(instance.SessionId)
	if err != nil {
		return err
	}
	dockerClient, err := d.factory.GetForSession(session)
	if err != nil {
		return err
	}
	return dockerClient.ContainerResize(instance.Name, rows, cols)
}

func (d *DinD) InstanceGetTerminal(instance *types.Instance) (net.Conn, error) {
	session, err := d.getSession(instance.SessionId)
	if err != nil {
		return nil, err
	}
	dockerClient, err := d.factory.GetForSession(session)
	if err != nil {
		return nil, err
	}
	return dockerClient.CreateAttachConnection(instance.Name)
}

func (d *DinD) InstanceUploadFromUrl(instance *types.Instance, fileName, dest, url string) error {
	log.Printf("Downloading file [%s]\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Could not download file [%s]. Error: %s\n", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Could not download file [%s]. Status code: %d\n", url, resp.StatusCode)
	}
	session, err := d.getSession(instance.SessionId)
	if err != nil {
		return err
	}
	dockerClient, err := d.factory.GetForSession(session)
	if err != nil {
		return err
	}

	copyErr := dockerClient.CopyToContainer(instance.Name, dest, fileName, resp.Body)

	if copyErr != nil {
		return fmt.Errorf("Error while downloading file [%s]. Error: %s\n", url, copyErr)
	}

	return nil
}

func (d *DinD) getInstanceCWD(instance *types.Instance) (string, error) {
	session, err := d.getSession(instance.SessionId)
	if err != nil {
		return "", err
	}
	dockerClient, err := d.factory.GetForSession(session)
	if err != nil {
		return "", err
	}
	b := bytes.NewBufferString("")

	if c, err := dockerClient.ExecAttach(instance.Name, []string{"bash", "-c", `pwdx $(</var/run/cwd)`}, b); c > 0 {
		return "", fmt.Errorf("Error %d trying to get CWD", c)
	} else if err != nil {
		return "", err
	}

	cwd := strings.TrimSpace(strings.Split(b.String(), ":")[1])

	return cwd, nil
}

func (d *DinD) InstanceUploadFromReader(instance *types.Instance, fileName, dest string, reader io.Reader) error {
	session, err := d.getSession(instance.SessionId)
	if err != nil {
		return err
	}
	dockerClient, err := d.factory.GetForSession(session)
	if err != nil {
		return err
	}
	var finalDest string
	if filepath.IsAbs(dest) {
		finalDest = dest
	} else {
		if cwd, err := d.getInstanceCWD(instance); err != nil {
			return err
		} else {
			finalDest = fmt.Sprintf("%s/%s", cwd, dest)
		}
	}

	copyErr := dockerClient.CopyToContainer(instance.Name, finalDest, fileName, reader)

	if copyErr != nil {
		return fmt.Errorf("Error while uploading file [%s]. Error: %s\n", fileName, copyErr)
	}

	return nil
}
