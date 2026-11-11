import { spawn } from 'node:child_process';
import { createInterface } from 'node:readline';

const child = spawn(process.execPath, ['node_modules/@sveltejs/mcp/dist/index.mjs'], {
	stdio: ['pipe', 'pipe', 'inherit']
});
const lines = createInterface({ input: child.stdout });
const expected = ['list-sections', 'get-documentation', 'svelte-autofixer'];
let succeeded = false;
const timeout = setTimeout(() => {
	console.error('MCP handshake timed out');
	child.kill();
}, 20_000);
function send(message) {
	child.stdin.write(`${JSON.stringify(message)}\n`);
}
lines.on('line', (line) => {
	let message;
	try {
		message = JSON.parse(line);
	} catch {
		return;
	}
	if (message.id === 1) {
		if (message.error) {
			console.error('MCP initialize failed');
			child.kill();
			return;
		}
		send({ jsonrpc: '2.0', method: 'notifications/initialized' });
		send({ jsonrpc: '2.0', id: 2, method: 'tools/list', params: {} });
	}
	if (message.id === 2) {
		const names = message.result?.tools?.map((tool) => tool.name) ?? [];
		succeeded = expected.every((name) => names.includes(name));
		console.log(
			succeeded
				? `Svelte MCP: initialize OK; tools: ${expected.join(', ')}`
				: 'MCP tools are missing'
		);
		child.kill();
	}
});
child.on('error', () => {
	console.error('Cannot start Svelte MCP');
});
child.on('close', () => {
	clearTimeout(timeout);
	lines.close();
	process.exitCode = succeeded ? 0 : 1;
});
send({
	jsonrpc: '2.0',
	id: 1,
	method: 'initialize',
	params: {
		protocolVersion: '2025-03-26',
		capabilities: {},
		clientInfo: { name: 'etheyl-check', version: '0.1.0' }
	}
});
