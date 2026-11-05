declare global {
  interface Window {
    // Defined by the inline script in index.html.
    reloadAfterAssetError(): void;
  }
}

export function reloadOnChunkLoadError(error: Error): never {
  window.reloadAfterAssetError();
  throw error;
}
