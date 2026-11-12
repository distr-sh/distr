import {
  CustomerOrganization,
  DeploymentRevisionStatus,
  DeploymentStatusType,
  DeploymentTarget,
  DeploymentWithLatestRevision,
} from '@distr-sh/distr-sdk';
import dayjs from 'dayjs';
import {toVendorHomeCustomers, toVendorHomeDeploymentHealth} from './vendor-home-mappers';

const fresh = () => dayjs().toISOString();
const old = () => dayjs().subtract(1, 'hour').toISOString();

const status = (type: DeploymentStatusType, createdAt = fresh()): DeploymentRevisionStatus => ({
  createdAt,
  deploymentRevisionId: 'revision',
  type,
  message: '',
});

const deployment = (
  latestStatus?: DeploymentRevisionStatus,
  currentStatus?: DeploymentRevisionStatus
): DeploymentWithLatestRevision => ({
  deploymentTargetId: 'target',
  application: {name: 'app', type: 'docker'},
  applicationId: 'app',
  applicationName: 'app',
  applicationVersionId: 'version',
  applicationVersionName: '1.0.0',
  applicationLink: '',
  latestStatus,
  currentStatus,
});

const customer = (id: string): CustomerOrganization => ({
  id,
  createdAt: fresh(),
  name: id,
  features: [],
});

const target = (
  deployments: DeploymentWithLatestRevision[],
  customerOrganization?: CustomerOrganization
): DeploymentTarget => ({
  name: 'target',
  type: 'docker',
  customerOrganization,
  deployments,
  metricsEnabled: false,
  imageCleanupEnabled: false,
  deploymentLogsEnabled: false,
});

describe('toVendorHomeDeploymentHealth', () => {
  it('should count a stale deployment as stale only and leave out progressing and unknown ones', () => {
    const targets = [
      target([deployment(status('healthy')), deployment(status('running')), deployment(status('error'))]),
      target([
        deployment(status('healthy', old())),
        deployment(status('progressing')),
        deployment(),
        deployment(status('healthy')),
      ]),
    ];
    expect(toVendorHomeDeploymentHealth(targets)).toEqual({total: 7, healthy: 2, running: 1, error: 1, stale: 1});
  });

  it('should judge a deployment by its applied revision, not by a failed update', () => {
    const failedUpdate = deployment(status('error'), status('healthy'));
    expect(toVendorHomeDeploymentHealth([target([failedUpdate])])).toEqual({
      total: 1,
      healthy: 1,
      running: 0,
      error: 0,
      stale: 0,
    });
  });
});

describe('toVendorHomeCustomers', () => {
  it('should count only the deployments of each customer', () => {
    const a = customer('a');
    const b = customer('b');
    const targets = [
      target([deployment(status('error'))], a),
      target([deployment(status('healthy')), deployment(status('running'))], a),
      target([deployment(status('error'))]),
      target([deployment(status('error'))], customer('other')),
    ];
    expect(toVendorHomeCustomers([a, b], targets).map((c) => [c.id, c.deploymentHealth])).toEqual([
      ['a', {total: 3, healthy: 1, running: 1, error: 1, stale: 0}],
      ['b', {total: 0, healthy: 0, running: 0, error: 0, stale: 0}],
    ]);
  });
});
