<script>
    import Welcome from 'www/lib/pages/welcome.svelte'
    import Counter from 'www/lib/pages/counter.svelte'
    let {page, ...data} = $props()
</script>

{#if "welcome" === page}
    <Welcome name={data.name} />
{:else if "counter" === page}
    <Counter />
{:else}
    <span>Not found.</span>
{/if}

