/*
Copyright 2018 The OpenEBS Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// TODO
// Rename this file by removing the version suffix information

package v1alpha1

const cstorPoolYamls = `
---
apiVersion: openebs.io/v1alpha1
kind: CASTemplate
metadata:
  name: cstor-pool-create-default
spec:
  defaultConfig:
  # CstorPoolImage is the container image that executes zpool replication and
  # communicates with cstor iscsi target
  - name: CstorPoolImage
    value: {{env "OPENEBS_IO_CSTOR_POOL_IMAGE" | default "openebs/cstor-pool:latest"}}
  # CstorPoolMgmtImage runs cstor pool and cstor volume replica related CRUD
  # operations
  - name: CstorPoolMgmtImage
    value: {{env "OPENEBS_IO_CSTOR_POOL_MGMT_IMAGE" | default "openebs/cstor-pool-mgmt:latest"}}
  # HostPathType is a hostPath volume i.e. mounts a file or directory from the
  # host node’s filesystem into a Pod. 'DirectoryOrCreate' value  ensures
  # nothing exists at the given path i.e. an empty directory will be created.
  - name: HostPathType
    value: DirectoryOrCreate
  # SparseDir is a hostPath directory where to look for sparse files
  - name: SparseDir
    value: {{env "OPENEBS_IO_CSTOR_POOL_SPARSE_DIR" | default "/var/openebs/sparse"}}
  # RunNamespace is the namespace where namespaced resources related to pool
  # will be placed
  - name: RunNamespace
    value: {{env "OPENEBS_NAMESPACE"}}
  # ServiceAccountName is the account name assigned to pool management pod
  # with permissions to view, create, edit, delete required custom resources
  - name: ServiceAccountName
    value: {{env "OPENEBS_SERVICE_ACCOUNT"}}
  # PoolResourceRequests allow you to specify resource requests that need to be available
  # before scheduling the containers. If not specified, the default is to use the limits
  # from PoolResourceLimits or the default requests set in the cluster. 
  - name: PoolResourceRequests
    value: "none"
  # PoolResourceLimits allow you to set the limits on memory and cpu for pool pods
  # The resource and limit value should be in the same format as expected by
  # Kubernetes. Example:
  #- name: PoolResourceLimits
  #  value: |-
  #      memory: 1Gi
  - name: PoolResourceLimits
    value: "none"
  # AuxResourceRequests allow you to set requests on side cars. Requests have to be specified
  # in the format expected by Kubernetes
  - name: AuxResourceRequests
    value: "none"
  # AuxResourceLimits allow you to set limits on side cars. Limits have to be specified
  # in the format expected by Kubernetes
  - name: AuxResourceLimits
    value: "none"
  # ResyncInterval specifies duration after which a controller should
  # resync the resource status
  - name: ResyncInterval
    value: "30"
  taskNamespace: {{env "OPENEBS_NAMESPACE"}}
  run:
    tasks:
    # Following are the list of run tasks executed in this order to
    # create a cstor storage pool
    - cstor-pool-create-putcstorpoolcr-default
    - cstor-pool-create-putcstorpooldeployment-default
    - cstor-pool-create-putstoragepoolcr-default
    - cstor-pool-create-patchstoragepoolclaim-default
---
apiVersion: openebs.io/v1alpha1
kind: CASTemplate
metadata:
  name: cstor-pool-delete-default
spec:
  defaultConfig:
    # RunNamespace is the namespace to use to delete pool resources
  - name: RunNamespace
    value: {{env "OPENEBS_NAMESPACE"}}
  taskNamespace: {{env "OPENEBS_NAMESPACE"}}
  run:
    tasks:
    # Following are run tasks executed in this order to delete a storage pool
    - cstor-pool-delete-listcstorpoolcr-default
    - cstor-pool-delete-deletecstorpoolcr-default
    - cstor-pool-delete-listcstorpooldeployment-default
    - cstor-pool-delete-deletecstorpooldeployment-default
    - cstor-pool-delete-liststoragepoolcr-default
    - cstor-pool-delete-deletestoragepoolcr-default
---
apiVersion: openebs.io/v1alpha1
kind: RunTask
metadata:
  name: cstor-pool-create-putcstorpoolcr-default
spec:
  meta: |
    apiVersion: openebs.io/v1alpha1
    kind: CStorPool
    action: put
    id: putcstorpoolcr
  post: |
    {{- jsonpath .JsonResult "{.metadata.name}" | trim | addTo "putcstorpoolcr.objectName" .TaskResult | noop -}}
    {{- jsonpath .JsonResult "{.metadata.uid}" | trim | addTo "putcstorpoolcr.objectUID" .TaskResult | noop -}}
    {{- jsonpath .JsonResult "{.metadata.labels.kubernetes\\.io/hostname}" | trim | addTo "putcstorpoolcr.nodeName" .TaskResult | noop -}}
  task: |-
    {{- $diskDeviceIdList:= .Storagepool.diskDeviceIdList }}
    apiVersion: openebs.io/v1alpha1
    kind: CStorPool
    metadata:
      name: {{.Storagepool.owner}}-{{randAlphaNum 4 |lower }}
      labels:
        openebs.io/storage-pool-claim: {{.Storagepool.owner}}
        kubernetes.io/hostname: {{.Storagepool.nodeName}}
        openebs.io/version: {{ .CAST.version }}
        openebs.io/cas-template-name: {{ .CAST.castName }}
    spec:
      disks:
        diskList:
        {{- range $k, $deviceID := $diskDeviceIdList }}
        - {{ $deviceID }}
        {{- end }}
      poolSpec:
        poolType: {{.Storagepool.poolType}}
        cacheFile: /tmp/{{.Storagepool.owner}}.cache
        overProvisioning: false
    status:
      phase: Init
---
apiVersion: openebs.io/v1alpha1
kind: RunTask
metadata:
  name: cstor-pool-create-putcstorpooldeployment-default
spec:
  meta: |
    runNamespace: {{.Config.RunNamespace.value}}
    apiVersion: extensions/v1beta1
    kind: Deployment
    action: put
    id: putcstorpooldeployment
  post: |
    {{- jsonpath .JsonResult "{.metadata.name}" | trim | addTo "putcstorpooldeployment.objectName" .TaskResult | noop -}}
  task: |-
    {{- $setResourceRequests := .Config.PoolResourceRequests.value | default "none" -}}
    {{- $resourceRequestsVal := fromYaml .Config.PoolResourceRequests.value -}}
    {{- $setResourceLimits := .Config.PoolResourceLimits.value | default "none" -}}
    {{- $resourceLimitsVal := fromYaml .Config.PoolResourceLimits.value -}}
    {{- $setAuxResourceRequests := .Config.AuxResourceRequests.value | default "none" -}}
    {{- $auxResourceRequestsVal := fromYaml .Config.AuxResourceRequests.value -}}
    {{- $setAuxResourceLimits := .Config.AuxResourceLimits.value | default "none" -}}
    {{- $auxResourceLimitsVal := fromYaml .Config.AuxResourceLimits.value -}}
    apiVersion: extensions/v1beta1
    kind: Deployment
    metadata:
      name: {{.TaskResult.putcstorpoolcr.objectName}}
      labels:
        openebs.io/storage-pool-claim: {{.Storagepool.owner}}
        openebs.io/cstor-pool: {{.TaskResult.putcstorpoolcr.objectName}}
        app: cstor-pool
        openebs.io/version: {{ .CAST.version }}
        openebs.io/cas-template-name: {{ .CAST.castName }}
    spec:
      strategy:
        type: Recreate
      replicas: 1
      selector:
        matchLabels:
          app: cstor-pool
      template:
        metadata:
          labels:
            app: cstor-pool
            openebs.io/storage-pool-claim: {{.Storagepool.owner}}
        spec:
          serviceAccountName: {{ .Config.ServiceAccountName.value }}
          nodeSelector:
            kubernetes.io/hostname: {{.Storagepool.nodeName}}
          containers:
          - name: cstor-pool
            image: {{ .Config.CstorPoolImage.value }}
            resources:
              {{- if ne $setResourceLimits "none" }}
              limits:
              {{- range $rKey, $rLimit := $resourceLimitsVal }}
                {{ $rKey }}: {{ $rLimit }}
              {{- end }}
              {{- end }}
              {{- if ne $setResourceRequests "none" }}
              requests:
              {{- range $rKey, $rReq := $resourceRequestsVal }}
                {{ $rKey }}: {{ $rReq }}
              {{- end }}
              {{- end }}
            ports:
            - containerPort: 12000
              protocol: TCP
            - containerPort: 3233
              protocol: TCP
            - containerPort: 3232
              protocol: TCP
            livenessProbe:
              exec:
                command:
                - /bin/sh
                - -c
                - zfs set io.openebs:livenesstimestap='$(date)' cstor-$OPENEBS_IO_CSTOR_ID
              failureThreshold: 3
              initialDelaySeconds: 300
              periodSeconds: 10
              timeoutSeconds: 30
            securityContext:
              privileged: true
            volumeMounts:
            - name: device
              mountPath: /dev
            - name: tmp
              mountPath: /tmp
            - name: sparse
              mountPath: {{ .Config.SparseDir.value }}
            - name: udev
              mountPath: /run/udev
            env:
              # OPENEBS_IO_CSTOR_ID env has UID of cStorPool CR.
            - name: OPENEBS_IO_CSTOR_ID
              value: {{.TaskResult.putcstorpoolcr.objectUID}}
              # To avoid clash between terminating and restarting pod
              # in case older zrepl gets deleted faster, we keep initial delay
            lifecycle:
              postStart:
                 exec:
                    command: ["/bin/sh", "-c", "sleep 2"]
          - name: cstor-pool-mgmt
            image: {{ .Config.CstorPoolMgmtImage.value }}
            resources:
              {{- if ne $setAuxResourceRequests "none" }}
              requests:
              {{- range $rKey, $rLimit := $auxResourceRequestsVal }}
                {{ $rKey }}: {{ $rLimit }}
              {{- end }}
              {{- end }}
              {{- if ne $setAuxResourceLimits "none" }}
              limits:
              {{- range $rKey, $rLimit := $auxResourceLimitsVal }}
                {{ $rKey }}: {{ $rLimit }}
              {{- end }}
              {{- end }}
            ports:
            - containerPort: 9500
              protocol: TCP
            securityContext:
              privileged: true
            volumeMounts:
            - name: device
              mountPath: /dev
            - name: tmp
              mountPath: /tmp
            - name: sparse
              mountPath: {{ .Config.SparseDir.value }}
            - name: udev
              mountPath: /run/udev
            env:
              # OPENEBS_IO_CSTOR_ID env has UID of cStorPool CR.
            - name: OPENEBS_IO_CSTOR_ID
              value: {{.TaskResult.putcstorpoolcr.objectUID}}
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: RESYNC_INTERVAL
              value: {{ .Config.ResyncInterval.value }}
          volumes:
          - name: device
            hostPath:
              # directory location on host
              path: /dev
              # this field is optional
              type: Directory
          - name: tmp
            hostPath:
              # From host, dir called /var/openebs/shared-<uid> is created to avoid clash if two replicas run on same node.
              path: /var/openebs/shared-{{.Storagepool.owner}}
              type: {{ .Config.HostPathType.value }}
          - name: sparse
            hostPath:
              path: {{ .Config.SparseDir.value }}
              type: {{ .Config.HostPathType.value }}
          - name: udev
            hostPath:
              path: /run/udev
              type: Directory
---
apiVersion: openebs.io/v1alpha1
kind: RunTask
metadata:
  name: cstor-pool-create-putstoragepoolcr-default
spec:
  meta: |
    apiVersion: openebs.io/v1alpha1
    kind: StoragePool
    action: put
    id: putstoragepool
  post: |
    {{- jsonpath .JsonResult "{.metadata.name}" | trim | addTo "putstoragepool.objectName" .TaskResult | noop -}}
  task: |-
    {{- $diskList:= .Storagepool.diskList }}
    apiVersion: openebs.io/v1alpha1
    kind: StoragePool
    metadata:
      name: {{.TaskResult.putcstorpooldeployment.objectName}}
      labels:
        openebs.io/storage-pool-claim: {{.Storagepool.owner}}
        openebs.io/cstor-pool: {{.TaskResult.putcstorpooldeployment.objectName}}
        openebs.io/cas-type: cstor
        kubernetes.io/hostname: {{ .Storagepool.nodeName}}
        openebs.io/version: {{ .CAST.version }}
        openebs.io/cas-template-name: {{ .CAST.castName }}
    spec:
      disks:
        diskList:
        {{- range $k, $diskName := $diskList }}
        - {{ $diskName }}
        {{- end }}
      poolSpec:
        poolType: {{.Storagepool.poolType}}
        cacheFile: /tmp/{{.Storagepool.owner}}.cache
        overProvisioning: false
---
apiVersion: openebs.io/v1alpha1
kind: RunTask
metadata:
  name: cstor-pool-create-patchstoragepoolclaim-default
spec:
  meta: |
    id: patchstoragepoolclaim
    apiVersion: openebs.io/v1alpha1
    kind: StoragePoolClaim
    objectName: {{.Storagepool.owner}}
    action: patch
  task: |-
    type: merge
    pspec: |-
      status:
        phase: Online
---
# This run task lists all cstor pool CRs that need to be deleted
apiVersion: openebs.io/v1alpha1
kind: RunTask
metadata:
  name: cstor-pool-delete-listcstorpoolcr-default
spec:
  meta: |
    id: listcstorpoolcr
    apiVersion: openebs.io/v1alpha1
    kind: CStorPool
    action: list
    options: |-
      labelSelector: openebs.io/storage-pool-claim={{.Storagepool.owner}}
  post: |
    {{- $csps := jsonpath .JsonResult "{range .items[*]}pkey=csps,{@.metadata.name}=;{end}" | trim | default "" | splitList ";" -}}
    {{- $csps | notFoundErr "cstor pool cr not found" | saveIf "listcstorpoolcr.notFoundErr" .TaskResult | noop -}}
    {{- $csps | keyMap "csplist" .ListItems | noop -}}
---
# This run task delete all the required cstor pool CR
apiVersion: openebs.io/v1alpha1
kind: RunTask
metadata:
  name: cstor-pool-delete-deletecstorpoolcr-default
spec:
  meta: |
    apiVersion: openebs.io/v1alpha1
    kind: CStorPool
    action: delete
    id: deletecstorpoolcr
    objectName: {{ keys .ListItems.csplist.csps | join "," }}
---
# This run task lists all the required cstor pool deployments that need to be deleted
apiVersion: openebs.io/v1alpha1
kind: RunTask
metadata:
  name: cstor-pool-delete-listcstorpooldeployment-default
spec:
  meta: |
    id: listcstorpooldeployment
    apiVersion: extensions/v1beta1
    runNamespace: {{.Config.RunNamespace.value}}
    kind: Deployment
    action: list
    options: |-
      labelSelector: openebs.io/storage-pool-claim={{.Storagepool.owner}}
  post: |
    {{- $csds := jsonpath .JsonResult "{range .items[*]}pkey=csds,{@.metadata.name}=;{end}" | trim | default "" | splitList ";" -}}
    {{- $csds | notFoundErr "cstor pool deployment not found" | saveIf "listcstorpooldeployment.notFoundErr" .TaskResult | noop -}}
    {{- $csds | keyMap "csdlist" .ListItems | noop -}}
---
# This run task deletes all the required cstor pool deployments
apiVersion: openebs.io/v1alpha1
kind: RunTask
metadata:
  name: cstor-pool-delete-deletecstorpooldeployment-default
spec:
  meta: |
    id: deletecstorpooldeployment
    runNamespace: {{.Config.RunNamespace.value}}
    apiVersion: extensions/v1beta1
    kind: Deployment
    action: delete
    objectName: {{ keys .ListItems.csdlist.csds | join "," }}
---
# This run task lists all storage pool CRs that need to be deleted
apiVersion: openebs.io/v1alpha1
kind: RunTask
metadata:
  name: cstor-pool-delete-liststoragepoolcr-default
spec:
  meta: |
    id: liststoragepoolcr
    apiVersion: openebs.io/v1alpha1
    kind: StoragePool
    action: list
    options: |-
      labelSelector: openebs.io/storage-pool-claim={{.Storagepool.owner}}
  post: |
    {{- $sps := jsonpath .JsonResult "{range .items[*]}pkey=sps,{@.metadata.name}=;{end}" | trim | default "" | splitList ";" -}}
    {{- $sps | notFoundErr "storge pool cr not found" | saveIf "listcstorpoolcr.notFoundErr" .TaskResult | noop -}}
    {{- $sps | keyMap "splist" .ListItems | noop -}}
---
# This run task deletes the required storagepool object
apiVersion: openebs.io/v1alpha1
kind: RunTask
metadata:
  name: cstor-pool-delete-deletestoragepoolcr-default
spec:
  meta: |
    id: deletestoragepoolcr
    apiVersion: openebs.io/v1alpha1
    kind: StoragePool
    action: delete
    objectName: {{ keys .ListItems.splist.sps | join "," }}
---
`

// CstorPoolArtifacts returns the cstor pool related artifacts corresponding to
// latest version
func CstorPoolArtifacts() (list artifactList) {
	list.Items = append(list.Items, ParseArtifactListFromMultipleYamls(cstorPools{})...)
	return
}

type cstorPools struct{}

// FetchYamls returns all the yamls related to cstor pool in a string
// format
//
// NOTE:
//  This is an implementation of MultiYamlFetcher
func (c cstorPools) FetchYamls() string {
	return cstorPoolYamls
}
