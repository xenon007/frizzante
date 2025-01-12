<script>
    let {from,...data} = $props()
</script>

{#await from then Component}
    <Component.default {...data}/>
{/await}