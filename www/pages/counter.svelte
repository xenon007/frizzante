<script>
    let counter = $state()
    function increase() {
        counter++
    }
</script>
<button onmouseup={increase}>Increase ({counter})</button>