<script lang="ts">
	let {
		type = 'error',
		message
	}: {
		type?: 'error' | 'success' | 'warning' | 'info';
		message: string;
	} = $props();

	const styles = {
		error: 'bg-red-50 border-red-200 text-red-700',
		success: 'bg-green-50 border-green-200 text-green-700',
		warning: 'bg-amber-50 border-amber-200 text-amber-700',
		info: 'bg-primary-50 border-primary-200 text-primary-700'
	};
</script>

<div class={`p-4 border rounded-lg text-sm mb-4 ${styles[type]}`}>{message}</div>