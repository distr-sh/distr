declare global {
  interface Window {
    // Defined by the inline script in index.html, which also handles failed entry bundles.
    reloadAfterAssetError(): boolean;
  }
}

export function reloadOnChunkLoadError(error: Error): Promise<never> {
  return window.reloadAfterAssetError() ? new Promise(() => {}) : Promise.reject(error);
}
