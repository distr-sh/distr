import {SupportBundleResourceSummary} from '../../types/support-bundle';
import {buildResourceTree} from './resource-tree';

function resource(name: string): SupportBundleResourceSummary {
  return {id: name, createdAt: '', name, contentType: 'text/plain'};
}

describe('buildResourceTree', () => {
  it('groups the names into folders in upload order and counts the files below each folder', () => {
    const tree = buildResourceTree([
      resource('namespaces/default/events.yaml'),
      resource('cluster/version.json'),
      resource('namespaces/default/pods/web/nginx'),
      resource('namespaces/kube-system/events.yaml'),
      resource('summary.yaml'),
    ]);

    expect(tree.fileCount).toBe(5);
    expect(tree.files.map((file) => file.name)).toEqual(['summary.yaml']);
    expect(tree.folders.map((folder) => folder.path)).toEqual(['namespaces', 'cluster']);

    const namespaces = tree.folders[0];
    expect(namespaces.fileCount).toBe(3);
    expect(namespaces.folders.map((folder) => folder.path)).toEqual(['namespaces/default', 'namespaces/kube-system']);

    const defaultNamespace = namespaces.folders[0];
    expect(defaultNamespace.files.map((file) => file.resource.name)).toEqual(['namespaces/default/events.yaml']);
    expect(defaultNamespace.folders[0].folders[0].files.map((file) => file.name)).toEqual(['nginx']);
  });

  it('keeps names without a folder at the top level and ignores empty segments', () => {
    const tree = buildResourceTree([resource('system-info'), resource('/logs//web')]);

    expect(tree.files.map((file) => file.name)).toEqual(['system-info']);
    expect(tree.folders.map((folder) => folder.path)).toEqual(['logs']);
    expect(tree.folders[0].files.map((file) => file.name)).toEqual(['web']);
  });
});
