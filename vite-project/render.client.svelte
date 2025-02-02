<script>
    <!--app-imports-->
    import {setContext} from "svelte";

    let {pageId, paths, data} = $props()
    let reactivePageId = $state(pageId)
    let reactiveData = $state({...data})
    setContext("data", reactiveData)
    setContext("page", page)
    setContext("pagePath", pagePath)
    window.addEventListener('popstate', (event) => {
        page(event.state.pageId)
    });

    function escapeRegExp(string) {
        return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    }

    function pagePath(pageId) {
        let path = paths[pageId] ?? ''
        if (!paths[pageId]) {
            return ""
        }
        const resolved = {}
        for (let key in data.path) {
            resolved[key] = false
        }
        for (let key in data.path) {
            const value = data[key]
            const regex = escapeRegExp(`{${key}}`)
            let oldPath = path
            path = path.replaceAll(new RegExp(regex, 'g'), value)
            resolved[key] = oldPath === path
        }

        return path
    }

    function page(pageId) {
        if (!paths[pageId]) {
            return
        }
        const path = pagePath(pageId)

        fetch(path, {headers: {"Accept": "application/json"}}).then(async (response) => {
            reactiveData = await response.json()
            history.pushState({pageId}, '', path);
            reactivePageId = pageId
        })
    }
</script>

<!--app-router-->