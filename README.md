# log-watcher

Log-watcher is an application that saves logs from pods, writes them to file to be collected by filebeat.


## Deployment

Simply download the `reasources.yaml` manifest file, change it to fit your needs and use `kubectl apply`.

## Changes to `resources.yaml`

It is needed to make changes to the filebeat container in the manifest file. [Tutorial from elastic.](https://www.elastic.co/guide/en/beats/filebeat/current/running-on-docker.html)
```yaml
- name: file-beat
        image: docker.elastic.co/beats/filebeat:8.2.2
        imagePullPolicy: IfNotPresent
        command: [] #filebeat setup command
        volumeMounts:
        - name: shared-logs
          mountPath: /tmp #where the files will be
        resources:
          requests:
            memory: 100Mi
            cpu: 1000m
          limits:
            memory: 500Mi
            cpu: 2000m
        securityContext:
          runAsUser: 1000
        env:
        #the config file

```
Change the NAME_FILTER value in the environment of log-watcher container to specify what pods you want to watch.

```yaml
env:
    - name: NAME_FILTER
      value: "hello-*"
```
