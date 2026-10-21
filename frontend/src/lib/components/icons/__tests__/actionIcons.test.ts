import { describe, it, expect } from 'vitest';
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { actionIconPaths, isActionIcon, ACTION_ICONS } from '../actionIcons';

const pagesDir = join(__dirname, '../../../../../../backend/definitions/pages');

describe('action icons', () => {
	it('finds icons by name, case-insensitive, and by alias', () => {
		expect(actionIconPaths('New')).toBe(ACTION_ICONS.New);
		expect(actionIconPaths('new')).toBe(ACTION_ICONS.New);
		expect(actionIconPaths('Add')).toBe(ACTION_ICONS.New);
		expect(actionIconPaths(undefined)).toBeNull(); // the bullet point
		expect(actionIconPaths('NoSuchIcon')).toBeNull();
	});

	it('every image of a page action is a known icon', () => {
		const files = readdirSync(pagesDir).filter((f) => f.endsWith('.yaml'));
		expect(files.length).toBeGreaterThan(0);
		const unknown: string[] = [];
		let found = 0;
		for (const file of files) {
			for (const m of readFileSync(join(pagesDir, file), 'utf8').matchAll(/^\s*image:\s*["']?([^"'\s#]+)/gm)) {
				found++;
				if (!isActionIcon(m[1])) unknown.push(`${file}: ${m[1]}`);
			}
		}
		expect(found).toBeGreaterThan(0);
		expect(unknown).toEqual([]);
	});
});
