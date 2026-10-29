import { describe, it, expect } from 'vitest';
import { errorDetailsToCsv } from '../errorDetails';

describe('errorDetailsToCsv', () => {
	it('writes each section with its caption, columns and rows', () => {
		const csv = errorDetailsToCsv({
			title: 'G/L entries that do not balance',
			sections: [
				{
					caption: 'Totals',
					columns: [{ caption: 'Document No.' }, { caption: 'Difference', numeric: true }],
					rows: [['G00001', '1 000,00']],
					marked: [0]
				},
				{
					caption: 'Entries',
					columns: [{ caption: 'Description' }],
					rows: [['Rent; "May"']]
				}
			]
		});
		expect(csv.split('\r\n')).toEqual([
			'G/L entries that do not balance',
			'',
			'Totals',
			'Document No.;Difference',
			'G00001;1 000,00',
			'',
			'Entries',
			'Description',
			'"Rent; ""May"""',
			''
		]);
	});
});
