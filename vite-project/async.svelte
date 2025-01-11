<script>
    let {from,...data} = $props()
</script>

{#await from then Component}
    <Component.default name="world" {...data}/>
{/await}