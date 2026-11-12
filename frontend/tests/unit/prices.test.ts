import assert from 'node:assert/strict';
import { test } from 'node:test';
import { rublesToKopecks } from '../../src/lib/creators/prices.ts';

test('RUB input preserves exact kopecks and accepts Russian and decimal point notation', () => {
	for (const [input, expected] of [
		['0', 0],
		['0,01', 1],
		['0.29', 29],
		['299,50', 29950],
		['499.01', 49901],
		[' 10,5 ', 1050],
		['1000000', 100000000]
	] as const) {
		assert.equal(rublesToKopecks(input), expected, input);
	}
	for (const input of [
		'',
		'-1',
		'1.001',
		'1e3',
		'Infinity',
		'NaN',
		'1000000,01',
		'1 000',
		'10,',
		'00.10'
	]) {
		assert.equal(rublesToKopecks(input), null, input);
	}
});
