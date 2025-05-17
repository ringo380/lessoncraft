# LessonCraft Kubernetes Deployment

This directory contains Kubernetes manifests for deploying LessonCraft to a Kubernetes cluster.

## Directory Structure

- **backup**: Contains manifests for MongoDB backup and restore operations
- **configmaps**: Contains ConfigMap resources for application configuration
- **deployments**: Contains Deployment resources for the application components
- **ingress**: Contains Ingress resources for external access
- **monitoring**: Contains manifests for monitoring with Prometheus and Grafana
- **secrets**: Contains Secret resources for sensitive configuration
- **services**: Contains Service resources for networking
- **storage**: Contains PersistentVolumeClaim resources for data persistence

## Deployment Instructions

### Prerequisites

- Kubernetes cluster (v1.22+)
- kubectl configured to access your cluster
- Storage class available for persistent volumes

### Deployment Steps

1. **Create namespace**:
   ```bash
   kubectl create namespace lessoncraft
   ```

2. **Apply storage resources**:
   ```bash
   kubectl apply -f storage/ -n lessoncraft
   ```

3. **Apply ConfigMaps and Secrets**:
   ```bash
   kubectl apply -f configmaps/ -n lessoncraft
   kubectl apply -f secrets/ -n lessoncraft
   ```

4. **Deploy MongoDB**:
   ```bash
   kubectl apply -f deployments/mongodb-deployment.yaml -n lessoncraft
   kubectl apply -f services/mongodb-service.yaml -n lessoncraft
   ```

5. **Deploy LessonCraft and L2**:
   ```bash
   kubectl apply -f deployments/lessoncraft-deployment.yaml -n lessoncraft
   kubectl apply -f deployments/l2-deployment.yaml -n lessoncraft
   kubectl apply -f services/lessoncraft-service.yaml -n lessoncraft
   kubectl apply -f services/l2-service.yaml -n lessoncraft
   ```

6. **Deploy monitoring**:
   ```bash
   kubectl apply -f monitoring/ -n lessoncraft
   ```

7. **Configure ingress**:
   ```bash
   kubectl apply -f ingress/ -n lessoncraft
   ```

8. **Set up backups**:
   ```bash
   kubectl apply -f backup/mongodb-backup-cronjob.yaml -n lessoncraft
   ```

## Backup and Restore

### Backup

MongoDB backups are automatically created by the CronJob defined in `backup/mongodb-backup-cronjob.yaml`. By default, backups are created daily at 2:00 AM and stored in the `mongodb-backup-pvc` PersistentVolumeClaim.

### Restore

To restore MongoDB from a backup:

1. Create the restore job:
   ```bash
   kubectl create -f backup/mongodb-restore-job.yaml -n lessoncraft
   ```

2. To restore from a specific backup, edit the job to set the `BACKUP_FILE` environment variable:
   ```bash
   kubectl edit job mongodb-restore -n lessoncraft
   ```

3. Monitor the restore job:
   ```bash
   kubectl logs job/mongodb-restore -n lessoncraft
   ```

## Monitoring

The deployment includes Prometheus and Grafana for monitoring:

- Prometheus: Collects metrics from the application
- Grafana: Provides visualization of metrics

Access Grafana at `http://grafana.lessoncraft.example.com` (adjust domain as needed).

Default credentials:
- Username: admin
- Password: admin (change this in production)

## Production Considerations

Before deploying to production:

1. **Security**:
   - Replace default credentials in Secret resources
   - Configure TLS for Ingress resources
   - Set up network policies

2. **Scaling**:
   - Adjust resource requests and limits based on expected load
   - Configure horizontal pod autoscaling

3. **High Availability**:
   - Deploy multiple replicas of stateless components
   - Configure pod anti-affinity rules

4. **Backup**:
   - Configure off-site backup storage
   - Test restore procedures regularly