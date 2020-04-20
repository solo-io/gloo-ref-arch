# Two-phased canary rollout with Gloo, part 4

In the last section, we implemented our two-phased canary workflow with a Helm chart that drastically reduced the amount
of config we needed to manage, down to just a set of Helm values. And it simplified the workflow down to a series of Helm 
upgrades. 

However, the chart was very limited and did not support a number of must-haves, including customizing the images, 
supporting custom paths for routes, supporting multiple routes, and enabling route options like traffic shadowing. 
In this part, we're going to generalize the chart to solve these things, making the Helm chart cover a much wider 
set of use cases and making it easier to for development teams to adopt. 

## Customizing the deployment

Let's introduce a few deployment customizations. Specifically, let's allow setting up replication for workloads, 
allow customizing the image and tag that are used for a particular subset, and support using other ports for traffic. 

### Replicas

To support replicas, we need to set a single value on the deployment object. We can expose that in our deployment 
values like so:

```yaml
deployment:
  v1: 
    replicas: 1
```

And we can access it in our deployment Helm template like this:

```yaml
{{- $relname := .Release.Name -}}
{{- $relns := .Release.Namespace -}}
{{- range $version, $config := .Values.deployment }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ $relname }}-{{ $version }}
  namespace: {{ $relns }}
spec:
  replicas: {{ $config.replicas }}
  selector:
    ...
```

### Customizing the image and args

Let's extract the image as another configuration on the deployment. We'll still assume a single container, and we'll 
assume it's available in a public registry that we don't need credentials to access. 

Here's how we might specify the image: 
```yaml
deployment: 
  v1:
    replicas: 1
    image: hashicorp/http-echo
    imagePullPolicy: Always
    args:
      - |-
          "-text=version:{{ .Release.Name }}-v1"
      - -listen=:8080
```

For now, we'll skip customizing ports and mounting certificates for TLS. However, notice that the args include a 
templated value. This arg needs to be passed into the `tpl` function to inform Helm that it should be rendered with the
required values, in this case the name of the release. We can update our deployment container with these values as follows:

```yaml
...
      containers:
        - image: {{ $config.image }}
          args:
            {{- range $arg := $config.args }}
            - {{ tpl $arg $ }}
            {{- end }}
          imagePullPolicy: {{ $config.imagePullPolicy }}
          name: {{ $relname }}-{{ $version }}
          ports:
            - containerPort: 8080
...
```

### Summary

We now have a deployment that will work for any single container service that runs on port 8080. While this will 
make our values file more verbose, it also makes our Helm chart significantly more flexible. Especially after we 
introduce customizable routes. 

## Routing

Currently, in our route table we only create a single route that matches on the release name as a prefix. Let's 
update this to support multiple different paths, that can be a prefix, exact, or regex match. In doing so, we want 
every route to implement the same two-phased canary workflow; in other words, when our two-phased canary workflow 
is invoked, we want it to be a single set of values that executes the workflow on all our routes.

### Modeling routes


