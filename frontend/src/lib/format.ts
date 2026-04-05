export function formatDate(iso: string): string {
	return new Date(iso).toLocaleDateString();
}

export function formatDateTime(iso: string): string {
	return new Date(iso).toLocaleString();
}

export function formatSpend(value: number): string {
	return `${value.toLocaleString()} bUEC`;
}

export function formatShares(value: number): string {
	if (!Number.isFinite(value)) return '0';
	if (Number.isInteger(value)) return value.toLocaleString();
	return value.toLocaleString(undefined, {
		minimumFractionDigits: 0,
		maximumFractionDigits: 4
	});
}

export function formatExpiry(iso: string): string {
	const date = new Date(iso);
	const now = new Date();
	const diffMs = date.getTime() - now.getTime();
	const diffDays = Math.ceil(diffMs / 86_400_000);
	if (diffDays <= 0) return 'Expired';
	if (diffDays === 1) return 'Expires tomorrow';
	if (diffDays <= 7) return `Expires in ${diffDays} days`;
	return `Expires ${date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}`;
}