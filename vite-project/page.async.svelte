<script module>
    let PreviousComponent = $state(false)
</script>
<script>
    let {from} = $props()
    from.then((page) => {
        PreviousComponent = page
    })
</script>

{#await from}
    {#if PreviousComponent}
        <PreviousComponent.default/>
    {/if}
{:then Component}
    <Component.default/>
{/await}