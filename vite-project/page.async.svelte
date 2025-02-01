<script module>
    let PreviousComponent = $state(false)
</script>
<script>
    let {from,...data} = $props()
    from.then((page)=>{
        PreviousComponent = page
    })
</script>

{#await from}
    {#if PreviousComponent}
        <PreviousComponent.default {...data}/>
    {/if}
{:then Component}
    <Component.default {...data}/>
{/await}