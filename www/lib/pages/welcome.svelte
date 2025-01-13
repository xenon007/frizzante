<script>
    import Layout from '$lib/components/layout.svelte'
    import {getContext} from "svelte";
    const data = getContext("data")
</script>

<svelte:head>
    <title>Welcome</title>
</svelte:head>

<Layout>
    <h1>Hello {data.name}.</h1>
</Layout>