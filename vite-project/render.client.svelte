<script>
    import Async from './async.svelte'
    import {setContext} from "svelte";
    let {pageId, paths: paths, data} = $props()
    let reactivePageId = $state(pageId)
    let reactiveData = $state({...data})
    setContext("data", reactiveData)
    setContext("page", page)
    setContext("path", path)
    window.addEventListener('popstate', (event) => {
        page(event.state.pageId)
    });
    function escapeRegExp(string) {
        return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    }
    function path(pageId){
        let path = paths[pageId]??''
        if (!paths[pageId]) {
            return ""
        }
        const resolved = {}
        for(let key in data.path) {
            resolved[key] = false
        }
        for(let key in data.path) {
            const value = data[key]
            const regex = escapeRegExp(`{${key}}`)
            let oldPath = path
            path = path.replaceAll(new RegExp(regex,'g'), value)
            resolved[key] = oldPath === path
        }

        return path
    }
    function page(pageId){
        if (!paths[pageId]) {
            return
        }
        history.pushState({pageId}, '', path(pageId));
        reactivePageId = pageId
    }
</script>

<!--app-router-->