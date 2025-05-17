package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ringo380/lessoncraft/config"
	"github.com/ringo380/lessoncraft/docker"
	"github.com/ringo380/lessoncraft/event"
	"github.com/ringo380/lessoncraft/handlers"
	"github.com/ringo380/lessoncraft/id"
	"github.com/ringo380/lessoncraft/k8s"
	"github.com/ringo380/lessoncraft/provisioner"
	"github.com/ringo380/lessoncraft/pwd"
	"github.com/ringo380/lessoncraft/pwd/types"
	"github.com/ringo380/lessoncraft/scheduler"
	"github.com/ringo380/lessoncraft/scheduler/task"
	"github.com/ringo380/lessoncraft/storage"

	"github.com/ringo380/lessoncraft/api"
	"github.com/ringo380/lessoncraft/api/store"
)

func main() {
	config.ParseFlags()

	// Initialize MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Error connecting to MongoDB: ", err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Fatal("Error disconnecting from MongoDB: ", err)
		}
	}()

	db := client.Database("lessoncraft")
	lessonStore := store.NewMongoLessonStore(db)

	// Initialize core LessonCraft components
	e := initEvent()
	s := initStorage()
	df := initDockerFactory(s)
	kf := initK8sFactory(s)

	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(df, s), provisioner.NewDinD(id.XIDGenerator{}, df, s))
	sp := provisioner.NewOverlaySessionProvisioner(df)

	core := pwd.NewLessonCraft(df, e, s, sp, ipf) // Using the new function name as per the TODO

	tasks := []scheduler.Task{
		task.NewCheckPorts(e, df),
		task.NewCheckSwarmPorts(e, df),
		task.NewCheckSwarmStatus(e, df),
		task.NewCollectStats(e, df, s),
		task.NewCheckK8sClusterStatus(e, kf),
		task.NewCheckK8sClusterExposedPorts(e, kf),
	}
	sch, err := scheduler.NewScheduler(tasks, s, e, core)
	if err != nil {
		log.Fatal("Error initializing the scheduler: ", err)
	}

	sch.Start()

	d, err := time.ParseDuration("4h")
	if err != nil {
		log.Fatalf("Cannot parse duration Got: %v", err)
	}

	playground := types.Playground{
		Domain:                      config.PlaygroundDomain,
		DefaultDinDInstanceImage:    "franela/dind",
		AvailableDinDInstanceImages: []string{"franela/dind"},
		AllowWindowsInstances:       config.NoWindows,
		DefaultSessionDuration:      d,
		Extras:                      map[string]interface{}{"LoginRedirect": "http://localhost:3000"},
		Privileged:                  true,
	}
	if _, err := core.PlaygroundNew(playground); err != nil {
		log.Fatalf("Cannot create default playground. Got: %v", err)
	}

	// Initialize API handlers
	router := mux.NewRouter()
	apiHandler := api.NewApiHandler(lessonStore)
	apiHandler.RegisterRoutes(router)

	// Bootstrap LessonCraft handlers
	handlers.Bootstrap(core, e)
	handlers.Register(router)

	// Start server
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal(err)
	}
}

func initStorage() storage.StorageApi {
	s, err := storage.NewFileStorage(config.SessionsFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal("Error initializing StorageAPI: ", err)
	}
	return s
}

func initEvent() event.EventApi {
	return event.NewLocalBroker()
}

func initDockerFactory(s storage.StorageApi) docker.FactoryApi {
	return docker.NewLocalCachedFactory(s)
}

func initK8sFactory(s storage.StorageApi) k8s.FactoryApi {
	return k8s.NewLocalCachedFactory(s)
}
