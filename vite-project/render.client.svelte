<script>
    import Async from './async.svelte'
    import {setContext} from "svelte";
    let {pageId, pagesToPaths, ...data} = $props()
    let reactivePageId = $state(pageId)
    let reactiveData = $state({...data})
    setContext("data", reactiveData)
    setContext("page", page)
    window.addEventListener('popstate', (event) => {
        page(event.state.pageId)
    });
    function escapeRegExp(string) {
        return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    }
    function page(pageId){
        if (!pagesToPaths[pageId]) {
            return
        }
        let path = pagesToPaths[pageId]
        const resolved = {}
        for(let key in data) {
            resolved[key] = false
        }
        for(let key in data) {
            const value = data[key]
            const regex = escapeRegExp(`{${key}}`)
            let oldPath = path
            path = path.replaceAll(new RegExp(regex,'g'), value)
            resolved[key] = oldPath === path
        }
        history.pushState({pageId}, '', path);
        reactivePageId = pageId
    }
</script>

<!--app-router-->