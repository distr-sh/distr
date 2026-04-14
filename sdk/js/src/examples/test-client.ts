import {Client, DistrService} from '../index';
import {clientConfig} from './config';

const client = new Client(clientConfig);
const distr = new DistrService(clientConfig);

try {
  let newDockerApp = await client.createApplication({
    type: 'docker',
    name: 'A Docker App',
  });
  log(newDockerApp, 'create docker application');

  newDockerApp.name = 'A Docker App (updated)';
  newDockerApp = await client.updateApplication(newDockerApp);
  log(newDockerApp, 'update docker application');

  const newDockerVersion = await client.createApplicationVersion(
    newDockerApp.id!,
    {
      name: 'v1.0.0',
    },
    {composeFile: 'hello: world'}
  );
  log(newDockerVersion, 'create docker application version');

  let newKubernetesApp = await client.createApplication({
    type: 'kubernetes',
    name: 'A Kubernetes App',
  });
  log(newKubernetesApp, 'create kubernetes application');

  const newKubernetesVersion = await client.createApplicationVersion(
    newKubernetesApp.id!,
    {
      name: 'v1.0.0',
      chartName: 'my-chart',
      chartVersion: '1.0.0',
      chartType: 'repository',
      chartUrl: 'https://my.chart.repo',
    },
    {templateFile: 'hello', baseValuesFile: 'base: values'}
  );
  log(newKubernetesVersion, 'create kubernetes application version');

  const applications = await client.getApplications();
  log(applications, 'get applications');
  for (let a of applications) {
    const app = await client.getApplication(a.id!);
    log(app, 'get application by id');
  }

  const newDockerDeploymentTarget = await client.createDeploymentTarget({
    name: 'A Docker Deployment Target',
    type: 'docker',
    deployments: [],
    metricsEnabled: false,
    imageCleanupEnabled: false,
    deploymentLogsEnabled: false,
  });
  log(newDockerDeploymentTarget, 'create docker deployment target');

  const newKubernetesDeploymentTarget = await client.createDeploymentTarget({
    name: 'A Kubernetes Deployment Target ',
    type: 'kubernetes',
    deployments: [],
    metricsEnabled: false,
    imageCleanupEnabled: false,
    deploymentLogsEnabled: false,
    namespace: 'glasskube',
    scope: 'namespace',
  });
  log(newKubernetesDeploymentTarget, 'create kubernetes deployment target');

  await client.createOrUpdateDeployment({
    applicationVersionId: newDockerVersion.id!,
    deploymentTargetId: newDockerDeploymentTarget.id!,
  });
  const recentlyDeployedTo = await client.getDeploymentTarget(newDockerDeploymentTarget.id!);
  log(recentlyDeployedTo, 'get recently deployed to');

  const deploymentTargets = await client.getDeploymentTargets();
  log(deploymentTargets, 'get deployment targets');
  for (let dt of deploymentTargets) {
    const deploymentTarget = await client.getDeploymentTarget(dt.id!);
    log(deploymentTarget, 'get deployment target by id');
  }

  const newDockerVersionV2 = await client.createApplicationVersion(
    newDockerApp.id!,
    {
      name: 'v2.0.0',
    },
    {composeFile: 'hello: worldv2'}
  );
  log(newDockerVersionV2, 'create docker application version v2.0.0');

  const updateAllResult = await distr.updateAllDeployments(newDockerApp.id!, newDockerVersionV2.id!);
  log(updateAllResult, 'update all deployments result');
} catch (error) {
  console.error(error);
}

function log(obj: any, title?: string) {
  if (title) {
    console.log(title);
  }
  console.log(JSON.stringify(obj, null, 2));
  console.log('-------------------');
}
