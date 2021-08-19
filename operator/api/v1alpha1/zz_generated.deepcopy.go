// +build !ignore_autogenerated

/*
Copyright 2020 In Loco Tecnologia da Informação S.A.

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionsRunner) DeepCopyInto(out *ActionsRunner) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionsRunner.
func (in *ActionsRunner) DeepCopy() *ActionsRunner {
	if in == nil {
		return nil
	}
	out := new(ActionsRunner)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ActionsRunner) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionsRunnerJob) DeepCopyInto(out *ActionsRunnerJob) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionsRunnerJob.
func (in *ActionsRunnerJob) DeepCopy() *ActionsRunnerJob {
	if in == nil {
		return nil
	}
	out := new(ActionsRunnerJob)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ActionsRunnerJob) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionsRunnerJobList) DeepCopyInto(out *ActionsRunnerJobList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ActionsRunnerJob, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionsRunnerJobList.
func (in *ActionsRunnerJobList) DeepCopy() *ActionsRunnerJobList {
	if in == nil {
		return nil
	}
	out := new(ActionsRunnerJobList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ActionsRunnerJobList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionsRunnerJobSpec) DeepCopyInto(out *ActionsRunnerJobSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionsRunnerJobSpec.
func (in *ActionsRunnerJobSpec) DeepCopy() *ActionsRunnerJobSpec {
	if in == nil {
		return nil
	}
	out := new(ActionsRunnerJobSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionsRunnerJobStatus) DeepCopyInto(out *ActionsRunnerJobStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionsRunnerJobStatus.
func (in *ActionsRunnerJobStatus) DeepCopy() *ActionsRunnerJobStatus {
	if in == nil {
		return nil
	}
	out := new(ActionsRunnerJobStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionsRunnerList) DeepCopyInto(out *ActionsRunnerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ActionsRunner, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionsRunnerList.
func (in *ActionsRunnerList) DeepCopy() *ActionsRunnerList {
	if in == nil {
		return nil
	}
	out := new(ActionsRunnerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ActionsRunnerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionsRunnerReplicaSet) DeepCopyInto(out *ActionsRunnerReplicaSet) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionsRunnerReplicaSet.
func (in *ActionsRunnerReplicaSet) DeepCopy() *ActionsRunnerReplicaSet {
	if in == nil {
		return nil
	}
	out := new(ActionsRunnerReplicaSet)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ActionsRunnerReplicaSet) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionsRunnerReplicaSetList) DeepCopyInto(out *ActionsRunnerReplicaSetList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ActionsRunnerReplicaSet, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionsRunnerReplicaSetList.
func (in *ActionsRunnerReplicaSetList) DeepCopy() *ActionsRunnerReplicaSetList {
	if in == nil {
		return nil
	}
	out := new(ActionsRunnerReplicaSetList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ActionsRunnerReplicaSetList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionsRunnerReplicaSetSpec) DeepCopyInto(out *ActionsRunnerReplicaSetSpec) {
	*out = *in
	in.Template.DeepCopyInto(&out.Template)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionsRunnerReplicaSetSpec.
func (in *ActionsRunnerReplicaSetSpec) DeepCopy() *ActionsRunnerReplicaSetSpec {
	if in == nil {
		return nil
	}
	out := new(ActionsRunnerReplicaSetSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionsRunnerReplicaSetStatus) DeepCopyInto(out *ActionsRunnerReplicaSetStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionsRunnerReplicaSetStatus.
func (in *ActionsRunnerReplicaSetStatus) DeepCopy() *ActionsRunnerReplicaSetStatus {
	if in == nil {
		return nil
	}
	out := new(ActionsRunnerReplicaSetStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionsRunnerRepository) DeepCopyInto(out *ActionsRunnerRepository) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionsRunnerRepository.
func (in *ActionsRunnerRepository) DeepCopy() *ActionsRunnerRepository {
	if in == nil {
		return nil
	}
	out := new(ActionsRunnerRepository)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionsRunnerSpec) DeepCopyInto(out *ActionsRunnerSpec) {
	*out = *in
	in.Metadata.DeepCopyInto(&out.Metadata)
	out.Repository = in.Repository
	if in.Capabilities != nil {
		in, out := &in.Capabilities, &out.Capabilities
		*out = make([]ActionsRunnerCapability, len(*in))
		copy(*out, *in)
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.VolumeMounts != nil {
		in, out := &in.VolumeMounts, &out.VolumeMounts
		*out = make([]v1.VolumeMount, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.EnvFrom != nil {
		in, out := &in.EnvFrom, &out.EnvFrom
		*out = make([]v1.EnvFromSource, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make(map[string]v1.ResourceRequirements, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionsRunnerSpec.
func (in *ActionsRunnerSpec) DeepCopy() *ActionsRunnerSpec {
	if in == nil {
		return nil
	}
	out := new(ActionsRunnerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActionsRunnerStatus) DeepCopyInto(out *ActionsRunnerStatus) {
	*out = *in
	in.LastProbeTime.DeepCopyInto(&out.LastProbeTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionsRunnerStatus.
func (in *ActionsRunnerStatus) DeepCopy() *ActionsRunnerStatus {
	if in == nil {
		return nil
	}
	out := new(ActionsRunnerStatus)
	in.DeepCopyInto(out)
	return out
}
