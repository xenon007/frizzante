<script>
    <!--app-imports-->
    import {setContext} from 'svelte'
    let {pageId, paths, data} = $props()
    let reactiveData = $state({...data})
    setContext("data", reactiveData)
    setContext("page", ()=>{})
    setContext("pagePath", function(pageId){
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
    })

</script>

<!--app-router-->