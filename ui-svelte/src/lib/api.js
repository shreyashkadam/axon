import axios from 'axios';

// The launcher runs on port 8080, which will serve our Svelte app.
// We can use relative paths for API calls.
const launcherUrl = '';

// --- Launcher Control API ---
export const getManagedNodes = () => axios.get(`${launcherUrl}/api/control/nodes`);
export const addNewNode = (nodeId) => axios.post(`${launcherUrl}/api/control/add`, { node_id: nodeId });
export const stopNode = (nodeId) => axios.post(`${launcherUrl}/api/control/stop/${nodeId}`);
export const startNode = (nodeId) => axios.post(`${launcherUrl}/api/control/start/${nodeId}`);
export const deleteNode = (nodeId) => axios.post(`${launcherUrl}/api/control/delete/${nodeId}`);

// --- KV Store Node API ---
// The Go backend needs to be configured with CORS to allow requests from the Svelte dev server port
export const getNodeApiUrl = (port) => `http://localhost:${port}`;
export const getAllKeys = (port) => axios.get(`${getNodeApiUrl(port)}/kv`);
export const getKey = (port, key) => axios.get(`${getNodeApiUrl(port)}/kv/${key}`);
export const putKey = (port, key, value) => axios.put(`${getNodeApiUrl(port)}/kv/${key}`, { value });
export const deleteKey = (port, key) => axios.delete(`${getNodeApiUrl(port)}/kv/${key}`);