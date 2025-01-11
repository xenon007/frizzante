<script>
    import WELCOME from './../../lib/pages/welcome.svelte'
    import COUNTER from './../../lib/pages/counter.svelte'
    let {page, ...data} = $props()
</script>
{#if 'welcome' === page}
    <WELCOME {...data} />
{:else if 'counter' === page}
    <COUNTER {...data} />
{/if}