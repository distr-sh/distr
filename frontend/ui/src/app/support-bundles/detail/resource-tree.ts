import {SupportBundleResourceSummary} from '../../types/support-bundle';

export interface SupportBundleResourceFolder {
  name: string;
  path: string;
  folders: SupportBundleResourceFolder[];
  files: SupportBundleResourceFile[];
  fileCount: number;
}

export interface SupportBundleResourceFile {
  name: string;
  resource: SupportBundleResourceSummary;
}

export function buildResourceTree(resources: SupportBundleResourceSummary[]): SupportBundleResourceFolder {
  const root = newFolder('', '');
  const folders = new Map<string, SupportBundleResourceFolder>();
  for (const resource of resources) {
    const segments = resource.name.split('/').filter((segment) => segment !== '');
    const name = segments.pop() ?? resource.name;
    let folder = root;
    folder.fileCount++;
    for (const segment of segments) {
      const path = folder.path === '' ? segment : `${folder.path}/${segment}`;
      let child = folders.get(path);
      if (!child) {
        child = newFolder(segment, path);
        folders.set(path, child);
        folder.folders.push(child);
      }
      child.fileCount++;
      folder = child;
    }
    folder.files.push({name, resource});
  }
  return root;
}

function newFolder(name: string, path: string): SupportBundleResourceFolder {
  return {name, path, folders: [], files: [], fileCount: 0};
}
