package tasks

import (
	"context"

	"github.com/openshift/cluster-monitoring-operator/pkg/client"
	"github.com/openshift/cluster-monitoring-operator/pkg/manifests"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MetricsServerTask struct {
	client    *client.Client
	enabled   bool
	ctx       context.Context
	factory   *manifests.Factory
	config    *manifests.Config
	namespace string
}

func NewMetricsServerTask(ctx context.Context, namespace string, client *client.Client, metricsServerEnabled bool, factory *manifests.Factory, config *manifests.Config) *MetricsServerTask {
	return &MetricsServerTask{
		client:    client,
		enabled:   metricsServerEnabled,
		factory:   factory,
		config:    config,
		namespace: namespace,
		ctx:       ctx,
	}
}

func (t *MetricsServerTask) Run(ctx context.Context) error {
	if t.enabled {
		return t.create(ctx)
	}
	return nil
}

func (t *MetricsServerTask) create(ctx context.Context) error {
	{
		sa, err := t.factory.MetricsServerServiceAccount()
		if err != nil {
			return errors.Wrap(err, "initializing MetricsServer ServiceAccount failed")
		}

		err = t.client.CreateOrUpdateServiceAccount(ctx, sa)
		if err != nil {
			return errors.Wrap(err, "reconciling MetricsServer ServiceAccount failed")
		}
	}
	{
		cr, err := t.factory.MetricsServerClusterRole()
		if err != nil {
			return errors.Wrap(err, "initializing metrics-server ClusterRolefailed")
		}

		err = t.client.CreateOrUpdateClusterRole(ctx, cr)
		if err != nil {
			return errors.Wrap(err, "reconciling metrics-server ClusterRole failed")
		}
	}
	{
		crb, err := t.factory.MetricsServerClusterRoleBinding()
		if err != nil {
			return errors.Wrap(err, "initializing MetricsServer ClusterRoleBinding failed")
		}

		err = t.client.CreateOrUpdateClusterRoleBinding(ctx, crb)
		if err != nil {
			return errors.Wrap(err, "reconciling MetricsServer ClusterRoleBinding failed")
		}
	}
	{
		crb, err := t.factory.MetricsServerClusterRoleBindingAuthDelegator()
		if err != nil {
			return errors.Wrap(err, "initializing metrics-server:system:auth-delegator ClusterRoleBinding failed")
		}

		err = t.client.CreateOrUpdateClusterRoleBinding(ctx, crb)
		if err != nil {
			return errors.Wrap(err, "reconciling metrics-server:system:auth-delegator ClusterRoleBinding failed")
		}
	}
	{
		rb, err := t.factory.MetricsServerRoleBindingAuthReader()
		if err != nil {
			return errors.Wrap(err, "initializing  metrics-server-auth-reader RoleBinding failed")
		}

		err = t.client.CreateOrUpdateRoleBinding(ctx, rb)
		if err != nil {
			return errors.Wrap(err, "reconciling  metrics-server-auth-reader RoleBinding failed")
		}
	}
	{
		s, err := t.factory.MetricsServerService()
		if err != nil {
			return errors.Wrap(err, "initializing MetricsServer Service failed")
		}

		err = t.client.CreateOrUpdateService(ctx, s)
		if err != nil {
			return errors.Wrap(err, "reconciling MetricsServer Service failed")
		}
	}
	{
		dep, err := t.factory.MetricsServerDeployment()
		if err != nil {
			return errors.Wrap(err, "initializing MetricsServer Deployment failed")
		}

		err = t.client.CreateOrUpdateDeployment(ctx, dep)
		if err != nil {
			return errors.Wrap(err, "reconciling MetricsServer Deployment failed")
		}
	}
	{
		sm, err := t.factory.MetricsServerServiceMonitor()
		if err != nil {
			return errors.Wrap(err, "initializing MetricsServer ServiceMonitors failed")
		}

		err = t.client.CreateOrUpdateServiceMonitor(ctx, sm)
		if err != nil {
			return errors.Wrapf(err, "reconciling %s/%s ServiceMonitor failed", sm.Namespace, sm.Name)
		}
	}
	{
		pdb, err := t.factory.MetricsServerPodDisruptionBudget()
		if err != nil {
			return errors.Wrap(err, "initializing MetricsServer PodDisruptionBudget failed")
		}

		if pdb != nil {
			err = t.client.CreateOrUpdatePodDisruptionBudget(ctx, pdb)
			if err != nil {
				return errors.Wrap(err, "reconciling MetricsServer PodDisruptionBudget failed")
			}
		}
	}
	{
		api, err := t.factory.MetricsServerAPIService()
		if err != nil {
			return errors.Wrap(err, "initializing MetricsServer APIService failed")
		}

		err = t.client.CreateOrUpdateAPIService(ctx, api)
		if err != nil {
			return errors.Wrap(err, "reconciling MetricsServer APIService failed")
		}
	}

	return t.removePrometheusAdapterResources(ctx)
}

func (t *MetricsServerTask) removePrometheusAdapterResources(ctx context.Context) error {
	pa := NewPrometheusAdapterTask(ctx, t.namespace, t.client, false, t.factory, t.config)
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-adapter",
			Namespace: t.namespace,
		},
	}

	{
		pdb, err := pa.factory.PrometheusAdapterPodDisruptionBudget()
		if err != nil {
			return errors.Wrap(err, "initializing PrometheusAdapter PodDisruptionBudget failed")
		}

		if pdb != nil {
			err = pa.client.DeletePodDisruptionBudget(ctx, pdb)
			if err != nil {
				return errors.Wrap(err, "deleting PrometheusAdapter PodDisruptionBudget failed")
			}
		}
	}
	{
		sm, err := pa.factory.PrometheusAdapterServiceMonitor()
		if err != nil {
			return errors.Wrap(err, "initializing PrometheusAdapter ServiceMonitors failed")
		}

		err = pa.client.DeleteServiceMonitor(ctx, sm)
		if err != nil {
			return errors.Wrapf(err, "deleting %s/%s ServiceMonitor failed", sm.Namespace, sm.Name)
		}
	}
	{
		s, err := pa.factory.PrometheusAdapterService()
		if err != nil {
			return errors.Wrap(err, "initializing PrometheusAdapter Service failed")
		}

		err = pa.client.DeleteService(ctx, s)
		if err != nil {
			return errors.Wrap(err, "deleting PrometheusAdapter Service failed")
		}
	}
	{
		err := pa.client.DeleteDeployment(ctx, d)
		if err != nil {
			return errors.Wrap(err, "deleting PrometheusAdapter Deployment failed")
		}
	}

	// TODO(slashpai): Add steps to remove other resources if any
	return nil
}
