# A simple Kubernetes Operator to return vSphere Virtual Machine information #

This repository contains a very simple Kubernetes Operator that uses VMware's __govmomi__ to return some simple virtual machine information through the status field of a __Custom Resource (CR)__, which is called ```VMInfo```. This will require us to extend Kubernetes with a new __Custom Resource Definition (CRD)__. The code shown here is for education purposes only, showing one way in which a Kubernetes controller / operator can access the underlying vSphere infrastructure for the purposes of querying resources.

You can think of a CRD as representing the desired state of a Kubernetes object or Custom Resource, and the function of the operator is to run the logic or code to make that desired state happen - in other words the operator has the logic to do whatever is necessary to achieve the object's desired state.

This is the second in the series of Kubernetes Operators to query the status of vSphere resources. The first was built to query ESXi resources (called __HostInfo__). Details about that operator can be found [here](https://github.com/cormachogan/hostinfo-operator).

## What are we going to do in this tutorial? ##

In this example, we will create a CRD called ```VMInfo```. VMInfo will contain the name of an virtual machine in its specification, possibly a Kubernetes Node. When a Custom Resource (CR) is created and subsequently queried, we will call an operator (logic in a controller) whereby some details about the virtual machine will be returned via the status fields of the object through govmomi API calls.

The following will be created as part of this tutorial:

* A __Customer Resource Definition (CRD)__
  * Group: ```Topology```
    * Kind: ```VMInfo```
    * Version: ```v1```
    * Specification will include a single item: ```Spec.Nodename```

* One or more __VMInfo Custom Resource / Object__ will be created through yaml manifests, each manifest containing the nodename of a virtual machine that we wish to query. The fields which will be updated to contain the relevant information from the VM (when the CR is queried) are:
  * ```Status.GuestId```
  * ```Status.TotalCPU```
  * ```Status.ResvdCPU```
  * ```Status.TotalMem```
  * ```Status.ResvdMem```
  * ```Status.PowerState```
  * ```Status.IPAddress```
  * ```Status.HWVersion```
  * ```Status.PathToVM```

* An __Operator__ (or business logic) to retrieve virtual machine information specified in the CR will be coded in the controller for this CR.

## What is not covered in this tutorial? ##

The assumption is that you already have a working Kubernetes cluster. Installation and deployment of a Kubernetes is outside the scope of this tutorial. If you do not have a Kubernetes cluster available, consider using __Kubernetes in Docker__ (shortened to __Kind__) which uses containers as Kubernetes nodes. A quickstart guide can be found here:

* [Kind (Kubernetes in Docker)](https://kind.sigs.K8s.io/docs/user/quick-start/)

The assumption is that you also have a __VMware vSphere environment__ comprising of at least one ESXi hypervisor with at least one virtual machine which is managed by a vCenter server. While the thought process is that your Kubernetes cluster will be running on vSphere infrastructure, and thus this operator will help you examine how the underlying vSphere resources are being consumed by the Kubernetes clusters running on top, it is not necessary for this to be the case for the purposes of this tutorial. You can use this code to query any vSphere environment from Kubernetes.

## What if I just want to understand some basic CRD concepts? ##

If this sounds even too daunting at this stage, I strongly recommend checking out the excellent tutorial on CRDs from my colleague, __Rafael Brito__. His [RockBand](https://github.com/brito-rafa/k8s-webhooks/blob/master/single-gvk/README.md) CRD tutorial uses some very simple concepts to explain how CRDs, CRs, Operators, spec and status fields work.

## Step 1 - Software Requirements ##

You will need the following components pre-installed on your desktop or workstation before we can build the CRD and operator.

* A __git__ client/command line
* [Go (v1.15+)](https://golang.org/dl/) - earlier versions may work but I used v1.15.
* [Docker Desktop](https://www.docker.com/products/docker-desktop)
* [Kubebuilder](https://go.kubebuilder.io/quick-start.html)
* [Kustomize](https://kubernetes-sigs.github.io/kustomize/installation/)
* Access to a Container Image Repositor (docker.io, quay.io, harbor)
* A __make__ binary - used by Kubebuilder

If you are interested in learning more about Golang basics, I found [this site](https://tour.golang.org/welcome/1) very helpful.

## Step 2 - KubeBuilder Scaffolding ##

The CRD is built using [kubebuilder](https://go.kubebuilder.io/).  I'm not going to spend a great deal of time talking about __KubeBuilder__. Suffice to say that KubeBuilder builds a directory structure containing all of the templates (or scaffolding) necessary for the creation of CRDs. Once this scaffolding is in place, this turorial will show you how to add your own specification fields and status fields, as well as how to add your own operator logic. In this example, our logic will login to vSphere, query and return virtual machine information via a Kubernetes CR / object / Kind called __VMInfo__, the values of which will be used to populate status fields in our CRs.

The following steps will create the scaffolding to get started.

```cmd
mkdir vminfo
$ cd vminfo
```

Next, define the Go module name of your CRD. In my case, I have called it __vminfo__. This creates a __go.mod__ file with the name of the module and the Go version (v1.15 here).

```cmd
$ go mod init vminfo
go: creating new go.mod: module vminfo
```

```cmd
$ ls
go.mod
```

```cmd
$ cat go.mod
module vminfo

go 1.15
```

Now we can proceed with building out the rest of the directory structure. The following __kubebuilder__ commands (__init__ and __create api__) creates all the scaffolding necessary to build our CRD and operator. You may choose an alternate __domain__ here if you wish. Simply make note of it as you will be referring to it later in the tutorial.

```cmd
kubebuilder init --domain corinternal.com
```

We must now define a resource. To do that, we again use kubebuilder to create the resource, specifying the API group, its version and supported kind. My API group is called __topology__, my kind is called __VMInfo__ and my initial version is __v1__.

```cmd
kubebuilder create api \
--group topology       \
--version v1           \
--kind VMInfo        \
--resource=true        \
--controller=true
```

The operator scaffolding (directory structure) is now in place. The next step is to define the specification and status fields in our CRD. After that, we create the controller logic which will watch our Custom Resources, and bring them to desired state (called a reconcile operation). More on this shortly.

## Step 3 - Create the CRD ##

Customer Resource Definitions [CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) are a way to extend Kubernetes through Custom Resources. We are going to extend a Kubernetes cluster with a new custom resource called __VMInfo__ which will retrieve information about the virtual machine whose name is specified in a Custom Resource. Thus, I will need to create a field called __nodename__ in the CRD - this defines the specification of the custom resource. We also add status fields, as these will be used to return information from the Virtual Machine.

This is done by modifying the __api/v1/vminfo_types.go__ file. Here is the initial scaffolding / template provided by kubebuilder:

```go
// VMInfoSpec defines the desired state of VMInfo
type HostInfoSpec struct {
        // INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
        // Important: Run "make" to regenerate code after modifying this file

        // Foo is an example field of VMInfo. Edit VMInfo_types.go to remove/update
        Foo string `json:"foo,omitempty"`
}

// VMInfoStatus defines the observed state of VMInfo
type VMInfoStatus struct {
        // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
        // Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
```

This file is modified to include a single __spec.nodename__ field and to return various __status__ fields. There are also a number of kubebuilder fields added, which are used to do validation and other kubebuilder related functions. The shortname "ch" will be used later on in our controller logic. Also, when we query any Custom Resources created with the CRD, e.g. ```kubectl get vminfo```, we want the output to display the nodename of the virtual machine.

Note that what we are doing here is for education purposes only. Typically what you would observe is that the spec and status fields would be similar, and it is the function of the controller to reconcile and differences between the two to achieve eventual consistency. But we are keeping things simple, as the purpose here is to show how vSphere can be queried from a Kubernetes Operator. Below is a snippet of the __vminfo_types.go__ showing the code changes. The code-complete [vminfo_types.go](api/v1/vminfo_types.go) is here.

```go
// VMInfoSpec defines the desired state of VMInfo
type VMInfoSpec struct {
        Nodename string `json:"nodename"`
}

// VMInfoStatus defines the observed state of VMInfo
type VMInfoStatus struct {
        GuestId    string `json:"guestId"`
        TotalCPU   int64  `json:"totalCPU"`
        ResvdCPU   int64  `json:"resvdCPU"`
        TotalMem   int64  `json:"totalMem"`
        ResvdMem   int64  `json:"resvdMem"`
        PowerState string `json:"powerState"`
        HwVersion  string `json:"hwVersion"`
        IpAddress  string `json:"ipAddress"`
        PathToVM   string `json:"pathToVM"`
}

// +kubebuilder:validation:Optional
// +kubebuilder:resource:shortName={"ch"}
// +kubebuilder:printcolumn:name="Nodename",type=string,JSONPath=`.spec.nodename`
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
```

We are now ready to create the CRD. There is one final step however, and this involves updating the __Makefile__ which kubebuilder has created for us. In the default Makefile created by kubebuilder, the following __CRD_OPTIONS__ line appears:

```Makefile
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
```

This CRD_OPTIONS entry should be changed to the following:

```Makefile
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:preserveUnknownFields=false,crdVersions=v1,trivialVersions=true"
```

Now we can build our CRD with the spec and status fields that we have place in the __api/v1/vminfo_types.go__ file.

```cmd
make manifests && make generate
```

## Step 4 - Install the CRD ##

The CRD is not currently installed in the Kubernetes Cluster.

```shell
$ kubectl get crd
NAME                                                               CREATED AT
antreaagentinfos.clusterinformation.antrea.tanzu.vmware.com        2020-11-18T17:14:03Z
antreacontrollerinfos.clusterinformation.antrea.tanzu.vmware.com   2020-11-18T17:14:03Z
clusternetworkpolicies.security.antrea.tanzu.vmware.com            2020-11-18T17:14:03Z
traceflows.ops.antrea.tanzu.vmware.com                             2020-11-18T17:14:03Z
```

To install the CRD, run the following make command:

```cmd
make install
```

Now check to see if the CRD is installed running the same command as before.

```shell
$ kubectl get crd
NAME                                                               CREATED AT
antreaagentinfos.clusterinformation.antrea.tanzu.vmware.com        2020-11-18T17:14:03Z
antreacontrollerinfos.clusterinformation.antrea.tanzu.vmware.com   2020-11-18T17:14:03Z
clusternetworkpolicies.security.antrea.tanzu.vmware.com            2020-11-18T17:14:03Z
traceflows.ops.antrea.tanzu.vmware.com                             2020-11-18T17:14:03Z
vminfoes.topology.corinternal.com                                  2021-01-18T11:25:20Z
```

Our new CRD ```vminfoes.topology.corinternal.com``` is now visible. Another useful way to check if the CRD has successfully deployed is to use the following command against our API group. Remember back in step 2 we specified the domain as ```corinternal.com``` and the group as ```topology```. Thus the command to query api-resources for this CRD is as follows:

```shell
$ kubectl api-resources --api-group=topology.corinternal.com
NAME         SHORTNAMES   APIGROUP                   NAMESPACED   KIND
vminfoes     ch           topology.corinternal.com   true           VMInfo
```

## Step 5 - Test the CRD ##

At this point, we can do a quick test to see if our CRD is in fact working. To do that, we can create a manifest file with a Custom Resource that uses our CRD, and see if we can instantiate such an object (or custom resource) on our Kubernetes cluster. Fortunately kubebuilder provides us with a sample manifest that we can use for this. It can be found in __config/samples__.

```shell
$ cd config/samples
$ ls
topology_v1_vminfo.yaml
```

```yaml
$ cat topology_v1_hostinfo.yaml
apiVersion: topology.corinternal.com/v1
kind: VMInfo
metadata:
  name: vminfo-sample
spec:
  # Add fields here
  foo: bar
```

We need to slightly modify this sample manifest so that the specification field matches what we added to our CRD. Note the spec: above where it states 'Add fields here'. We have removed the __foo__ field and added a __spec.nodename__ field, as per the __api/v1/vminfo_types.go__ modification earlier. Thus, after a simple modification, the CR manifest looks like this, where __tkg-cluster-1-18-5b-workers-kc5xn-dd68c4685-5v298__ is the name of the virtual machine that we wish to query. It is in fact a Tanzu Kubernetes worker node. It could be any virtual machine in your vSphere infrastructure.

```yaml
$ cat topology_v1_hostinfo.yaml
apiVersion: topology.corinternal.com/v1
kind: VMInfo
metadata:
  name: tkg-worker-1
spec:
  # Add fields here
  nodename: tkg-cluster-1-18-5b-workers-kc5xn-dd68c4685-5v298
```

To see if it works, we need to create this VMInfo Custom Resource.

```shell
$ kubectl create -f topology_v1_vminfo.yaml
vminfo.topology.corinternal.com/tkg-worker-1 created
```

```shell
$ kubectl get vminfo
NAME           NODENAME
tkg-worker-1   tkg-cluster-1-18-5b-workers-kc5xn-dd68c4685-5v298
```

Note that the nodename field is also printed, as per the kubebuilder directive that we placed in the __api/v1/vminfo_types.go__. As a final test, we will display the CR in yaml format.

```yaml
$ kubectl get vminfo -o yaml
apiVersion: v1
items:
- apiVersion: topology.corinternal.com/v1
  kind: VMInfo
  metadata:
    creationTimestamp: "2021-01-18T12:20:45Z"
    generation: 1
    managedFields:
    - apiVersion: topology.corinternal.com/v1
      fieldsType: FieldsV1
      fieldsV1:
        f:spec:
          .: {}
          f:nodename: {}
      manager: kubectl
      operation: Update
      time: "2021-01-18T12:20:45Z"
    - apiVersion: topology.corinternal.com/v1
      fieldsType: FieldsV1
      fieldsV1:
        f:status:
          .: {}
          f:guestId: {}
          f:hwVersion: {}
          f:ipAddress: {}
          f:pathToVM: {}
          f:powerState: {}
          f:resvdCPU: {}
          f:resvdMem: {}
          f:totalCPU: {}
          f:totalMem: {}
      manager: manager
      operation: Update
      time: "2021-01-18T12:20:46Z"
    name: tkg-worker-1
    namespace: default
    resourceVersion: "28841720"
    selfLink: /apis/topology.corinternal.com/v1/namespaces/default/vminfoes/tkg-worker-1
    uid: 2c60b273-a866-4344-baf5-0b3b924b65a5
  spec:
    nodename: tkg-cluster-1-18-5b-workers-kc5xn-dd68c4685-5v298
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

## Step 6 - Create the controller / manager ##

This appears to be working as expected. However there are no __Status__ fields displayed with our VM information in the __yaml__ output above. To see this information, we need to implement our operator / controller logic to do this. The controller implements the desired business logic. In this controller, we first read the vCenter server credentials from a Kubernetes secret passed to the controller (which we will create shortly). We will then open a session to my vCenter server, and get a list of virtual machines that it manages. We will then look for the virtual machine that is specified in the __spec.nodename__ field in the CR, and retrieve various information for this virtual machine. Finally we will update the appropriate status fields with this information, and we should be able to query it using the __kubectl get vminfo -o yaml__ command seen previously.

__Note:__ As has been pointed out, this code is not very optomized, and logging into vCenter Server for every reconcile request is not ideal. The login function should be moved out of the reconcile request, and it is something I will look at going forward. But for our present learning purposes, its fine to do this as we won't be overloading the vCenter Server with our handful of reconcile requests.

Once all this business logic has been added in the controller, we will need to be able to run it in the Kubernetes cluster. To achieve this, we will build a container image to run the controller logic. This will be provisioned in the Kubernetes cluster using a Deployment manifest. The deployment contains a single Pod that runs the container (it is called __manager__). The deployment ensures that my Pod is restarted in the event of a failure.

This is what kubebuilder provides as controller scaffolding - it is found in __controllers/vminfo_controller.go__ - we are most interested in the __VMInfoReconciler__ function:

```go
func (r *VMInfoReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
        _ = context.Background()
        _ = r.Log.WithValues("vminfo", req.NamespacedName)

        // your logic here

        return ctrl.Result{}, nil
}
```

Considering the business logic that I described above, this is what my updated __VMInfoReconciler__ function looks like. Hopefully the comments make is easy to understand, but at the end of the day, when this controller gets a reconcile request (something as simple as a get command will trigger this), the status fields in the Custom Resource are updated for the specific VM in the spec.nodename field. Note that I have omitted a number of required imports that also need to be added to the controller. Refer to the code for the complete [__vminfo_controller.go__](./controllers/vminfo_controller.go) code. One thing to note is that I am enabling insecure logins by default. This is something that you may wish to change in your code.

```go
func (r *VMInfoReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

        ctx := context.Background()
        log := r.Log.WithValues("vminfo", req.NamespacedName)

        ch := &topologyv1.VMInfo{}
        if err := r.Client.Get(ctx, req.NamespacedName, ch); err != nil {
                // add some debug information if it's not a NotFound error
                if !k8serr.IsNotFound(err) {
                        log.Error(err, "unable to fetch VMInfo")
                }
                return ctrl.Result{}, client.IgnoreNotFound(err)
        }

        msg := fmt.Sprintf("received reconcile request for %q (namespace: %q)", ch.GetName(), ch.GetNamespace())
        log.Info(msg)

        // We will retrieve these environment variables through passing 'secret' parameters via the manager manifest

        vc := os.Getenv("GOVMOMI_URL")
        user := os.Getenv("GOVMOMI_USERNAME")
        pwd := os.Getenv("GOVMOMI_PASSWORD")

        //
        // Create a vSphere/vCenter client
        //
        //    The govmomi client requires a URL object, u, not just a string representation of the vCenter URL.

        u, err := soap.ParseURL(vc)

        if err != nil {
                msg := fmt.Sprintf("unable to parse vCenter URL: error %s", err)
                log.Info(msg)
                return ctrl.Result{}, err
        }

        u.User = url.UserPassword(user, pwd)

        //
        // Ripped from https://github.com/vmware/govmomi/blob/master/examples/examples.go
        //

        // Share govc's session cache
        s := &cache.Session{
                URL:      u,
                Insecure: true,
        }

        c := new(vim25.Client)

        err = s.Login(ctx, c, nil)

        if err != nil {
                msg := fmt.Sprintf("unable to login to vCenter: error %s", err)
                log.Info(msg)
                return ctrl.Result{}, err
        }

        //
        // Create a view manager
        //

        m := view.NewManager(c)

        //
        // Create a container view of VirtualMachine objects
        //

        v, err := m.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)

        if err != nil {
                msg := fmt.Sprintf("unable to create container view for VirtualMachines: error %s", err)
                log.Info(msg)
                return ctrl.Result{}, err
        }

        defer v.Destroy(ctx)

        //
        // Retrieve summary property for all VMs
        //

        var vms []mo.VirtualMachine

        err = v.Retrieve(ctx, []string{"VirtualMachine"}, []string{"summary"}, &vms)

        if err != nil {
                msg := fmt.Sprintf("unable to retrieve VM summary: error %s", err)
                log.Info(msg)
                return ctrl.Result{}, err
        }

        //
        // Print summary for host in VMInfo specification info
        //

        for _, vm := range vms {
                if vm.Summary.Config.Name == ch.Spec.Nodename {
                        ch.Status.GuestId = string(vm.Summary.Guest.GuestId)
                        ch.Status.TotalCPU = int64(vm.Summary.Config.NumCpu)
                        ch.Status.ResvdCPU = int64(vm.Summary.Config.CpuReservation)
                        ch.Status.TotalMem = int64(vm.Summary.Config.MemorySizeMB)
                        ch.Status.ResvdMem = int64(vm.Summary.Config.MemoryReservation)
                        ch.Status.PowerState = string(vm.Summary.Runtime.PowerState)
                        ch.Status.HwVersion = string(vm.Summary.Guest.HwVersion)
                        ch.Status.IpAddress = string(vm.Summary.Guest.IpAddress)
                        ch.Status.PathToVM = string(vm.Summary.Config.VmPathName)
                }
        }

        if err := r.Status().Update(ctx, ch); err != nil {
                log.Error(err, "unable to update VMInfo status")
                return ctrl.Result{}, err
        }

        return ctrl.Result{}, nil
}
```

With the controller logic now in place, we can now proceed to build the controller / manager.

## Step 7 - Build the controller ##

At this point everything is in place to enable us to deploy the controller to the Kubernete cluster. If you remember back to the prerequisites in step 1, we said that you need access to a container image registry, such as docker.io or quay.io, or VMware's own [Harbor](https://github.com/goharbor/harbor/blob/master/README.md) registry. This is where we need this access to a registry, as we need to push the controller's container image somewhere that can be accessed from your Kubernetes cluster.

The __Dockerfile__ with the appropriate directives is already in place to build the container image and include the controller / manager logic. This was once again taken care of by kubebuilder. You must ensure that you login to your image repository, i.e. docker login, before proceeding with the __make__ commands. In this case, I am using the quay.io repository, e.g.

```shell
$ docker login quay.io
Username: cormachogan
Password: ***********
WARNING! Your password will be stored unencrypted in /home/cormac/.docker/config.json.
Configure a credential helper to remove this warning. See
https://docs.docker.com/engine/reference/commandline/login/#credentials-store

Login Succeeded
$
```

Next, set an environment variable called __IMG__ to point to your container image repository along with the name and version of the container image, e.g:

```shell
export IMG=quay.io/cormachogan/vminfo-controller:v1
```

Next, to create the container image of the controller / manager, and push it to the image container repository in a single step, run the following __make__ command. You could of course run this as two seperate commands as well, ```make docker-build``` followed by ```make docker-push``` if you so wished.

```shell
make docker-build docker-push IMG=quay.io/cormachogan/vminfo-controller:v1
```

The container image of the controller is now built and pushed to the container image registry. But we have not yet deployed it. We have to do one or two further modifications before we take that step.

## Step 8 - Modify the Manager manifest to include environment variables ##

Kubebuilder provides a manager manifest scaffold file for deploying the controller. However, since we need to provide vCenter details to our controller, we need to add these to the controller/manager manifest file. This is found in __config/manager/manager.yaml__. This manifest contains the deployment for the controller. In the spec, we need to add an additional __spec.env__ section which has the environment variables defined, as well as the name of our __secret__ (which we will create shortly). Below is a snippet of that code. Here is the code-complete [config/manager/manager.yaml](./config/manager/manager.yaml)).

```yaml
    spec:
      .
      .
        env:
          - name: GOVMOMI_USERNAME
            valueFrom:
              secretKeyRef:
                name: vc-creds
                key: GOVMOMI_USERNAME
          - name: GOVMOMI_PASSWORD
            valueFrom:
              secretKeyRef:
                name: vc-creds
                key: GOVMOMI_PASSWORD
          - name: GOVMOMI_URL
            valueFrom:
              secretKeyRef:
                name: vc-creds
                key: GOVMOMI_URL
      volumes:
        - name: vc-creds
          secret:
            secretName: vc-creds
      terminationGracePeriodSeconds: 10
```

Note that the __secret__, called __vc-creds__ above, contains the vCenter credentials. This secret needs to be deployed in the same namespace that the controller is going to run in, which is __vminfo-system__. Thus, the namespace and secret are created using the following commands, with the environment modified to your own vSphere infrastructure obviously:

```shell
$ kubectl create ns vminfo-system
namespace/vminfo-system created
```

```shell
$ kubectl create secret generic vc-creds \
--from-literal='GOVMOMI_USERNAME=administrator@vsphere.local' \
--from-literal='GOVMOMI_PASSWORD=VMware123!' \
--from-literal='GOVMOMI_URL=192.168.0.100' \
-n vminfo-system
secret/vc-creds created
```

We are now ready to deploy the controller to the Kubernetes cluster.

## Step 9 - Deploy the controller ##

To deploy the controller, we run another __make__ command. This will take care of all of the RBAC, cluster roles and role bindings necessary to run the controller, as well as pinging up the correct image, etc.

```shell
make deploy IMG=quay.io/cormachogan/vminfo-controller:v1
```

## Step 10 - Check controller functionality ##

Now that our controller has been deployed, let's see if it is working. There are a few different commands that we can run to verify the operator is working.

### Step 10.1 - Check the deployment and replicaset ###

The deployment should be READY. Remember to specify the namespace correctly when checking it.

```shell
$ kubectl get rs -n vminfo-system
NAME                                   DESIRED   CURRENT   READY   AGE
vminfo-controller-manager-79d6756854   1         1         1       37m

$ kubectl get deploy -n vminfo-system
NAME                          READY   UP-TO-DATE   AVAILABLE   AGE
vminfo-controller-manager     1/1     1            1           37m
```

### Step 10.2 - Check the Pods ###

The deployment manages a single controller Pod. There should be 2 containers READY in the controller Pod. One is the __controller / manager__ and the other is the __kube-rbac-proxy__. The [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy/blob/master/README.md) is a small HTTP proxy that can perform RBAC authorization against the Kubernetes API. It restricts requests to authorized Pods only.

```shell
$ kubectl get pods -n vminfo-system
NAME                                           READY   STATUS    RESTARTS   AGE
vminfo-controller-manager-79d6756854-b8jdq     2/2     Running   0          72s
```

If you experience issues with the one of the pods not coming online, use the following command to display the Pod status and examine the events.

```shell
kubectl describe pod vminfo-controller-manager-79d6756854-b8jdq -n vminfo-system
```

### Step 10.3 - Check the controller / manager logs ###

If we query the __logs__ on the manager container, we should be able to observe successful startup messages as well as successful reconcile requests from the HostInfo CR that we already deployed back in step 5. These reconcile requests should update the __Status__ fields with CPU information as per our controller logic. The command to query the manager container logs in the controller Pod is as follows:

```shell
kubectl logs vminfo-controller-manager-79d6756854-b8jdq -n vminfo-system manager
```

### Step 10.4 - Check if CPU statistics are returned in the status ###

Last but not least, let's see if we can see the CPU information in the __status__ fields of the HostInfo object created earlier.

```yaml
$ kubectl get vminfo tkg-worker-1 -o yaml
apiVersion: topology.corinternal.com/v1
kind: VMInfo
metadata:
  creationTimestamp: "2021-01-18T12:20:45Z"
  generation: 1
  managedFields:
  - apiVersion: topology.corinternal.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        .: {}
        f:nodename: {}
    manager: kubectl
    operation: Update
    time: "2021-01-18T12:20:45Z"
  - apiVersion: topology.corinternal.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:status:
        .: {}
        f:guestId: {}
        f:hwVersion: {}
        f:ipAddress: {}
        f:pathToVM: {}
        f:powerState: {}
        f:resvdCPU: {}
        f:resvdMem: {}
        f:totalCPU: {}
        f:totalMem: {}
    manager: manager
    operation: Update
    time: "2021-01-18T12:20:46Z"
  name: tkg-worker-1
  namespace: default
  resourceVersion: "28841720"
  selfLink: /apis/topology.corinternal.com/v1/namespaces/default/vminfoes/tkg-worker-1
  uid: 2c60b273-a866-4344-baf5-0b3b924b65a5
spec:
  nodename: tkg-cluster-1-18-5b-workers-kc5xn-dd68c4685-5v298
status:
  guestId: vmwarePhoton64Guest
  hwVersion: vmx-17
  ipAddress: 10.27.62.45
  pathToVM: '[vsanDatastore] 4d56b55f-11db-8822-6463-246e962f4914/tkg-cluster-1-18-5b-workers-kc5xn-dd68c4685-5v298.vmx'
  powerState: poweredOn
  resvdCPU: 0
  resvdMem: 0
  totalCPU: 2
```

__Success!!!__ Note that the output above is showing various status fields as per our business logic implemented in the controller. How cool is that? You can now go ahead and create additional __VMInfo__ manifests for different virtual machines in your vSphere environment managed by your vCenter server by specifying different nodenames in the manifest spec, and all you to get status from those VMs as well.

## Cleanup ##

To remove the __vminfo__ CR, operator and CRD, run the following commands.

### Remove the HostInfo CR ###

```shell
$ kubectl delete vminfo tkg-worker-1
vminfo.topology.corinternal.com "tkg-worker-1" deleted
```

### Removed the Operator/Controller deployment ###

Deleting the deployment will removed the ReplicaSet and Pods associated with the controller.

```shell
$ kubectl get deploy -n vminfo-system
NAME                          READY   UP-TO-DATE   AVAILABLE   AGE
vminfo-controller-manager   1/1     1            1           2d8h
```

```shell
$ kubectl delete deploy vminfo-controller-manager -n vminfo-system
deployment.apps "vminfo-controller-manager" deleted
```

### Remove the CRD ###

Next, remove the Custom Resource Definition, __vminfoes.topology.corinternal.com__.

```shell
$ kubectl get crds
NAME                                                               CREATED AT
antreaagentinfos.clusterinformation.antrea.tanzu.vmware.com        2021-01-14T16:31:58Z
antreacontrollerinfos.clusterinformation.antrea.tanzu.vmware.com   2021-01-14T16:31:58Z
clusternetworkpolicies.security.antrea.tanzu.vmware.com            2021-01-14T16:31:59Z
vminfoes.topology.corinternal.com                                2021-01-14T16:52:11Z
traceflows.ops.antrea.tanzu.vmware.com                             2021-01-14T16:31:59Z
```

```Makefile
$ make uninstall
go: creating new go.mod: module tmp
go: found sigs.k8s.io/controller-tools/cmd/controller-gen in sigs.k8s.io/controller-tools v0.2.5
/home/cormac/go/bin/controller-gen "crd:preserveUnknownFields=false,crdVersions=v1,trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
kustomize build config/crd | kubectl delete -f -
customresourcedefinition.apiextensions.k8s.io "vminfoes.topology.corinternal.com" deleted
```

```shell
$ kubectl get crds
NAME                                                               CREATED AT
antreaagentinfos.clusterinformation.antrea.tanzu.vmware.com        2021-01-14T16:31:58Z
antreacontrollerinfos.clusterinformation.antrea.tanzu.vmware.com   2021-01-14T16:31:58Z
clusternetworkpolicies.security.antrea.tanzu.vmware.com            2021-01-14T16:31:59Z
traceflows.ops.antrea.tanzu.vmware.com                             2021-01-14T16:31:59Z
```

The CRD is now removed. At this point, you can also delete the namespace created for the exercise, in this case __vminfo-system__. Removing this namespace will also remove the __vc_creds__ secret created earlier.

## What next? ##

One thing you could do it to extend the __VMInfo__ fields and Operator logic so that it returns even more information about the virtual machine. . There is a lot of information that can be retrieved via the govmomi __VirtualMachine__ API call.

You can now use __kusomtize__ to package the CRD and controller and distribute it to other Kubernetes clusters. Simply point the __kustomize build__ command at the location of the __kustomize.yaml__ file which is in __config/default__.

```shell
kustomize build config/default/ >> /tmp/vminfo.yaml
```

This newly created __vminfo.yaml__ manifest includes the CRD, RBAC, Service and Deployment for rolling out the operator on other Kubernetes clusters. Nice, eh?
