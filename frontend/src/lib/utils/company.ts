// A company as listed by GET /api/auth/companies. `name` is the technical key (it
// prefixes the company's tables and is what the session stores); `display_name` is
// what users see (BC/NAV Display Name) and may be blank.
export interface CompanyInfo {
	name: string;
	display_name: string;
}

// The name to show for a company: its display name, else its technical name.
export function companyLabel(company: CompanyInfo | undefined, fallback = ''): string {
	if (!company) return fallback;
	return company.display_name?.trim() || company.name;
}

// Label for a technical company name, looked up in a loaded company list.
export function companyLabelFor(companies: CompanyInfo[], name: string): string {
	return companyLabel(
		companies.find((c) => c.name === name),
		name
	);
}

// Case-insensitive filter on display name and technical name.
export function filterCompanies(companies: CompanyInfo[], filter: string): CompanyInfo[] {
	const q = filter.trim().toLowerCase();
	if (!q) return companies;
	return companies.filter(
		(c) => c.name.toLowerCase().includes(q) || (c.display_name ?? '').toLowerCase().includes(q)
	);
}
