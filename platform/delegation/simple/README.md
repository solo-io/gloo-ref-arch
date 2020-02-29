_This doc was automatically created by Valet 0.4.3-4-gc835eeb from the workflow defined in workflow.yaml. To deploy the demo, you can use `valet ensure -f workflow.yaml` from this directory, or execute the steps manually. Do not modify this file directly, it will be overwritten the next time the docs are generated._

# Extending a Monlithic Application with Gloo

In this workflow, we'll set up the petclinic application, which consists of a backend server and a database. Once this application is configured in Gloo, we'll look at how you can utilize **delegation** to enable teams to manage their own routes while an admin can manage the overall domain.


This workflow assumes you already have a Kubernetes cluster, and you've installed Gloo Enterprise to the gloo-system namespace.


 



 



 



```
kubectl apply -f routetable.yaml
```

 



```
kubectl apply -f vs-2.yaml
```

 



 

