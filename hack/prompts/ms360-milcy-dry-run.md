
What is the impact of deploying the following microservice on kagent namespace ?
```yaml
apiVersion: app.amdocs.com/v1beta1
kind: Microservice
metadata:
  name: shopping-cart
  namespace: kagent
  annotations:
    fabric8.io/iconUrl: "img/icons/spring-boot.svg"
    git-commit: "@git.commit.id@"
  labels:
    account: sprint
spec:
  version: 0.1.1-0066
  domain: shopping-cart
  component: funtional-backend
  partOf: OC12
  instance: ms-shopping-cart
  minReplicas: 1
  maxReplicas: 5
  minReadySeconds: 10
  terminationGracePeriodSeconds: 5
  revisionHistoryLimit: 15
  progressDeadlineSeconds: 120
  zeroReplica: false
  deploymentStrategy:
    strategyType: RollingUpdate
    rollingUpdate:
      maxSurge: 35%
      maxUnavailable: 1
  metrics:
    - type: Resource
      resource:
        name: cpu
        targetAverageUtilization: 50
  javaOptions:
    additionalOpts: -XX:+UseSerialGC
    initialHeapSize: 512m
    maxHeapSize: 812m
    maxRam: 8024m
    maxMetaspaceSize: 256m
  jvmDebug: true
  containerName: shoppingcart-experience-service-container
  exposePublicAPI: true
  swaggerConfig:
    - configmap://shopping-cart1-apigw-configmap
    - configmap://shopping-cart2-apigw-configmap
  # publicAPIUse: ExposeOnly
  externalServices:
    - name: customeraccessor-service
      api: /applicationContext/v0/entityAccessor/
      path: services.customerAccessor.url
      domain: shopping-cart
    - name: product-offfering-svc
      api: catalogManagement/v3.0
      path: services.productOfferingList.url
      domain: productorder
    - name: oh-service1
      api: /applicationContext/v0/entityAccessor/
      path: com.amdocs.orderworkflow.orchestrator.config.steps.ActivateLogicalResource.url
      domain: shopping-cart
    - name: oh-service2
      api: catalogManagement/v3.0
      path: com.amdocs.orderworkflow.orchestrator.config.steps.ActivateLogicalResource.amendUrl
  template:
    spec:
      imagePullSecrets:
        - name: amdocs-nexus-pull-secret
        - name: ms-lifecycle-operator-pull-secret
      containers:
        - name: shoppingcart-experience-service-container
          image: illin4261.corp.amdocs.com:28090/platform/busybox:1.36.0
          imagePullPolicy: IfNotPresent
          livenessProbe: # https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
            exec:
              command:
                - cat
                - /tmp/healthy
            initialDelaySeconds: 5
            periodSeconds: 5
            timeoutSeconds: 1
            successThreshold: 1
            failureThreshold: 3
          args:
            - /bin/sh
            - -c
            - touch /tmp/healthy; sleep 1000;
          env:
            - name: TIM_CUSTOM_ENV_VAR
              value: "1001"
          volumeMounts:
            - mountPath: /test-config
              name: test-volume
      volumes:
        - name: test-volume
          emptyDir: { }
```