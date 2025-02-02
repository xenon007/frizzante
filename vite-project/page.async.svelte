<script module>
    let PreviousComponent = $state(false)
</script>
<script>
    import {getContext, setContext} from "svelte";

    let {from, pageId} = $props()
    let data = getContext("data")

    const pagePath = getContext("pagePath")
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