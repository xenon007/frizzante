<script>
    import Async from './async.svelte'
    let {page, ...data} = $props()
</script>

{#if "welcome" === page}
    <Async from={import('$lib/pages/welcome.svelte')} {...data} />
{:else if "counter" === page}
    <Async from={import('$lib/pages/counter.svelte')} {...data} />
{:else}
    <span>Not found.</span>
{/if}

