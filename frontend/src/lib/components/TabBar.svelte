<!--
  Reusable tab bar. Replaces the duplicated border-b-2 button pattern in admin, mod, markets.

  Usage:
    <TabBar
      tabs={[
        { id: 'one', label: 'One' },
        { id: 'two', label: 'Two', badge: 3 }
      ]}
      bind:active={activeTab}
      onTabChange={(id) => { if (id === 'analytics') loadAnalytics(); }}
    />
-->
<script lang="ts">
	let {
		tabs,
		active = $bindable(),
		onTabChange
	}: {
		tabs: { id: string; label: string; badge?: number }[];
		active: string;
		onTabChange?: (id: string) => void;
	} = $props();

	function select(id: string) {
		active = id;
		onTabChange?.(id);
	}
</script>

<div class="flex flex-wrap gap-1 mb-6" role="tablist">
	{#each tabs as tab (tab.id)}
		<button
			role="tab"
			aria-selected={active === tab.id}
			onclick={() => select(tab.id)}
			class="px-4 py-2 rounded-full text-xs font-bold uppercase tracking-widest transition-colors
			       {active === tab.id
			         ? 'bg-primary-500 text-white shadow-sm'
			         : 'text-surface-500 hover:text-primary-500'}"
		>
			{tab.label}
			{#if tab.badge !== undefined && tab.badge > 0}
				<span class="ml-1.5 px-1.5 py-0.5 rounded-full text-[10px] font-bold
				             {active === tab.id ? 'bg-white/30 text-white' : 'bg-primary-500/15 text-primary-600'}">
					{tab.badge}
				</span>
			{/if}
		</button>
	{/each}
</div>
