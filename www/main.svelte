<script>
    import Greeting from '$lib/components/greeting.svelte'
    const { name } = $props()
</script>
<Greeting {name} />