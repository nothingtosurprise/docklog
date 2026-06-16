import { dockerEnabled, kubernetesEnabled } from './sharedState';

export function logsRouteForPod(podOrKey) {
  const p =
    typeof podOrKey === 'string'
      ? podOrKey
      : `${podOrKey.namespace}/${podOrKey.name}`;
  const query = { p };
  if (dockerEnabled() && kubernetesEnabled()) {
    query.rt = 'kubernetes';
  }
  return { path: '/logs', query };
}

export function logsRouteForContainer(containerId) {
  const query = { c: containerId };
  if (dockerEnabled() && kubernetesEnabled()) {
    query.rt = 'docker';
  }
  return { path: '/logs', query };
}

export function parsePodKey(key) {
  const slash = String(key || '').indexOf('/');
  if (slash <= 0 || slash >= key.length - 1) return null;
  return {
    namespace: key.slice(0, slash),
    name: key.slice(slash + 1),
  };
}
