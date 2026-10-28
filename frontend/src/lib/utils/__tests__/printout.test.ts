import { describe, it, expect } from 'vitest';
import { printoutOf, pdfBlob } from '../printout';

describe('printoutOf', () => {
	it('takes the PDF, the file name and the output of a job result', () => {
		expect(printoutOf({ pdf: 'JVBERi0=', filename: 'Sales Quote SQ1.pdf', output: 'preview' })).toEqual({
			pdf: 'JVBERi0=',
			filename: 'Sales Quote SQ1.pdf',
			output: 'preview'
		});
		expect(printoutOf({ pdf: 'JVBERi0=', output: 'print' })?.output).toBe('print');
	});

	it('downloads when the output is not known, with a default name', () => {
		expect(printoutOf({ pdf: 'JVBERi0=' })).toEqual({ pdf: 'JVBERi0=', filename: 'document.pdf', output: 'pdf' });
		expect(printoutOf({ pdf: 'JVBERi0=', output: 'printer' })?.output).toBe('pdf');
	});

	it('is null without a PDF', () => {
		expect(printoutOf(undefined)).toBeNull();
		expect(printoutOf({ refresh: true })).toBeNull();
		expect(printoutOf({ pdf: '' })).toBeNull();
	});
});

describe('pdfBlob', () => {
	it('decodes base64, also with a data URL prefix', () => {
		for (const text of ['JVBERi0=', 'data:application/pdf;base64,JVBERi0=']) {
			const blob = pdfBlob(text);
			expect(blob.type).toBe('application/pdf');
			expect(blob.size).toBe(5); // %PDF-
		}
	});
});
