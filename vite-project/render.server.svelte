<script>
    <!--app-imports-->
    import {setContext} from 'svelte'
    let {pageId, paths, path, data} = $props()
    let reactiveData = $state({...data})
    setContext("data", reactiveData)
    setContext("page", ()=>{})
    setContext("pagePath", function(pageId){
        let pathLocal = paths[pageId]??''
        if (!paths[pageId]) {
            return ""
        }
        const resolved = {}
        for(let key in path) {
            resolved[key] = false
        }
        for(let key in path) {
            const value = data[key]
            const regex = escapeRegExp(`{${key}}`)
            let oldPath = pathLocal
            pathLocal = pathLocal.replaceAll(new RegExp(regex,'g'), value)
            resolved[key] = oldPath === pathLocal
        }
        return pathLocal
    })

</script>

<!--app-router-->