<script>
    import Async from './async.svelte'
    let {page, ...data} = $props()
</script>
{#if 'welcome' === page}
    <Async from={import('./../../lib/pages/welcome.svelte')} {...data} />
{:else if 'about' === page}
    <Async from={import('./../../lib/pages/about.svelte')} {...data} />
{/if}