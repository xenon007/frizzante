<script>
    <!--app-imports-->
    import {setContext} from "svelte";

    let {pageId: serverPageId, paths, path, data: serverData} = $props()
    let pageId = $state(serverPageId)
    let data = $state({...serverData})
    setContext("data", data)
    setContext("page", page)
    setContext("pagePath", pagePath)
    window.addEventListener('popstate', (event) => {
        page(event.state.pageId)
    });

    function escapeRegExp(string) {
        return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    }

    function pagePath(pageId) {
        let pathLocal = paths[pageId] ?? ''
        if (!paths[pageId]) {
            return ""
        }
        const resolved = {}
        for (let key in path) {
            resolved[key] = false
        }
        for (let key in path) {
            const value = data[key]
            const regex = escapeRegExp(`{${key}}`)
            let oldPath = pathLocal
            pathLocal = pathLocal.replaceAll(new RegExp(regex, 'g'), value)
            resolved[key] = oldPath === pathLocal
        }

        return pathLocal
    }

    function page(pageIdLocal) {
        if (!paths[pageIdLocal]) {
            return
        }

        const path = pagePath(pageIdLocal)
        history.pushState({pageIdLocal}, '', path);
        pageId = pageIdLocal

        fetch(path, {headers: {"Accept": "application/json"}}).then(async (response) => {
            const newData = await response.json()
            
            for (const key in data) {
                delete data[key]
            }

            for (const key in newData) {
                data[key] = newData[key]
            }
        })
    }
</script>

<!--app-router-->