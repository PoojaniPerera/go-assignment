#Devops task solution

## Environment Set Up
### Pre-requisites

ubuntu vm : Entire solution has to be run on a UBUNtu Vm so it is preferable to develop and test entire solution on a ubuntu vm
kubernetes : To deploy and manage application and dependencies. I will be using minikube for my solution. If you are looking to deploy this in production environment you can go for a cloud provide. ex : eks registry in AWS service.
docker :  To package the application and its dependencies to a container image.
helm : kuberenetes package manager. This can be used to deploy a statefullset redi cluster easily.
dockerhub account to store docker images
kubectl configured : This will come with minikube.
You can set up your environment by installing these pre-requisites in your ubuntu vm .

## Architechture 

This go lang and redis application has to be deployed in a kubenetes cluster with high availability in mind. Design architechure will include following components.
* Two instances of go web service
* statefullset redis cluster with master and slave nodes. This statefullset redis master-slave replication will provide redundancy and high availability for the redis instance.
* Ingress controller to route traffice from outside the cluster . This will provide external access to the go web service. You can refer with <domain name>:<nodeport specfied in go-redis-app.yml kubernetes deployment/ service manifest file . nodeport is specified in service section >
* You can use a Load balancer if you are deploying in cloud kubernetes cluster to improve the high availability of this solution. 
* To password protect the redis cluster you can specify a kubernetes secret containing redis password. To use it inside the go app we can give it as an environment variable in go web service k8 deployment file.  If we use kubernetes manifest files from scratch to deploy redis cluster without helm then you wil have to create a config map containing redis.conf file details and you will have to mount it in your redis container in /etc/conf/redis.conf path . This file should contain requirepass key with your password value to passsword protect your redis cluster.
Now you only have to override auth.existingSecret and auth.existingSecretPasswordKey with yor helm install bitnami/redis command to override redis helm chart values for password protection.
* we will need to package the go webservice and its dependencies to a docker container  and deploy in k8 using minikube.



                  +-------------+
                  |    Redis    |
                  +-------------+
                  /       \
                 /         \
                /           \
+-------------+             +-------------+
|  Golang App |             |  Golang App |
+-------------+             +-------------+


## Solution steps

1. Developer steps (These are not necessary)
   Run below command to create go.mod init file 
   
> go mod init <unique module path ex : github.com/PoojaniPerera/ go-redis-app-test > 

2. include main.go file in your project directory

3. Run go get . command or go mod tidy command to download all required go modules (dependencies) for your go web sevice. This will create go.sum file as well.

4. Create a Dockerfile to  package go web service and go module dependencies in to a docker image . This docker image is used to easily deploy application to a container-based infrastructure like k8. Docker file is further explain with comments. We will not create a dockerimage for rediscluster as  we can get already built images from docker hub or if we are using helm charts it is already specified . Generally available image will be good for this solution thus we will not create a separate image for redis containers.

```Dockerfile
FROM golang:latest as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download 
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o main .
# EXPOSE 80
# ENTRYPOINT ["./main"]


FROM gcr.io/distroless/base-debian11 
COPY --from=builder /app/main .
EXPOSE 80
CMD ["/main"]
  
#multistage build lightweight one redis password protect search
  

```
> **_NOTE:_** Developer step ( not necessary)
After creating Dockerfile you can create a docker-compose.yml file to test your web service with one command without much hassel. This is good for developing and testing scenarios. 
docker-compose up

5. Build the docker image. Tag it and push to dockerhub registry.

> docker build -t my-go-redis-app.

> docker tag my-go-redis-app poojani/my-go-redis-app:v1

> docker login

> docker push poojani/my-go-redis-app:v1


6. Create required deployment and service yaml files (k8 manifest ) for go web service to deploy in in k8. Configure these files accordingly. You can configure to create replicas of the application to enable high availability and add loadbalancing for services. ex: go-redis-app.yml 

```yml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-redis-app
spec:
  replicas: 2
  selector:
    matchLabels:
      app: go-redis-app
  template:
    metadata:
      labels:
        app: go-redis-app
    spec:
      containers:
      - name: go-redis-app
        image: poojani/my-go-redis-app:v1
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8080
        env:
        - name: REDIS_URL
          valueFrom:
             configMapKeyRef:
              name: app-config
              key: redis-service
        - name: REDIS_HOST
          valueFrom:
             configMapKeyRef:
              name: app-config
              key: redis-host
        - name: REDIS_PORT
          value: "6379"
        - name: REDIS_PASSWORD
          valueFrom:
             secretKeyRef:
              name: redis-secret
              key: redis-password
        - name: REDIS_DB
          value: "0"
        - name: PORT
          value: "8080"
         


---
apiVersion: v1
kind: Service
metadata:
  name: go-redis-app-service
spec:
  selector:
    app: go-redis-app
  ports:
  - name: http
    port: 80
    targetPort: 8080
    nodePort: 30000        
  type: NodePort


```

7. Since we are using helm chart we would not need deployment and service files for redis cluster. But if we are not using helm this is needed and i have included files to create statefulset redis cluster without helm as well in my solution as an addition step.

8. create another yml file with kind : secret to create a kuberenetes secret  to store redis password. redis-secret.yml . Password has to be bas64 encoded before configuring in this kubernetes secret file. All secure configuration data can be stored in kubernetes secrets to improve security in your application. 

```yml
apiVersion: v1
kind: Secret
metadata:
  name: redis-secret
type: Opaque
data:
  redis-password: <base64encoded password>
```

9. create a yml file with kind configMap to  store other required configuratins for this go-redis web service. This is used because it would be a hassel to update redis_host kind of information inside the server if it changes. So config map is maintained to store such non-confidential data. 

**_app-config.yml_**
```yml
apiVersion: v1
kind: ConfigMap
metadata:
    name: app-config
data:
    redis-service: go-redis-app-master:6379
    redis-host: go-redis-app-master

```

10. For deploying in K8 i will be using minikube so now you will have to start minikue. (assuming docker engine is already started)

> minikube start

11. We will be using helmcharts to create statefullset master-slave redis cluster. Fisrt we will need to add bitnami/redis charts to our repo.

>  helm repo add bitnami https://charts.bitnami.com/bitnami

10. Create a yaml file to overide neccessary values when your are creating redis cluster with helm charts. you can pass this file to helm install command with --values tag

```yaml
replica:
    replicaCount: 1
    persistence:
        enabled: false 


networkPolicy:
  enabled: false # change networkPolicy enabled to 'false' from 'true'. This enables the app to discover the cluster easily.
  allowExternal: true # changed networkPolicy allowExternal from 'false' to 'true'. 

auth:
  existingSecret: redis-secret # mentioned app-secret to retrieve the redis-password we had initially created in step 2
  existingSecretPasswordKey: redis-password # mentioned the key which hold the base64 encoded hash of the password.
  
master:
    persistence:
        enabled: false # changed persistence enabled to false. minikube doesn't support persistence for redis. 
```

11. Create k8 secret you configured with kubectl apply command

> kubectl apply -f deployments/redis-secret.yaml

12. Run helm install command to create redis statefullset master-slave cluster.

> helm install go-redis-app bitnami/redis --values deployments/values-redis.yml

13. create k8 configMap

> kubectl apply -f deployments/app-config.yml 

14.  craete go app deployment and service

> kubectl apply -f deployments/go-redis-app.yml

15. check your cluster

> kubectl get all 

16. expose service 
