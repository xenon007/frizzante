<script>
    import ABOUT from './../../lib/pages/about.svelte'
    import WELCOME from './../../lib/pages/welcome.svelte'
    let {page, ...data} = $props()
</script>
{#if 'welcome' === page}
    <WELCOME {...data} />
{:else if 'about' === page}
    <ABOUT {...data} />
{/if}