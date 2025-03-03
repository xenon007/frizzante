import {getContext} from "svelte";

/**
 * @typedef Props
 * @property {import("svelte").Snippet} children
 * @property {"get" | "post" | "GET" | "POST"} [method]
 * @property {Record<string,string|number|boolean>} [form]
 * @property {"start"|"center"|"end"} [align]
 */

/**
 * @param {string} queryString
 */
function done(queryString) {
    /**
     * @param {Response} response
     */
    return function (response) {
        if (response.status >= 300) {
            console.error(`Submit request failed with status ${response.status} ${response.statusText}.`)
            return
        }

        response.json()
            .then(function (data) {
                const dataState = getContext("data")
                const state = window.history.state ?? {}
                history.replaceState(state, "", `${window.location.pathname}${document.location.hash}${queryString}`)
                for (const key in dataState) {
                    delete dataState[key]
                }

                for (const key in data) {
                    dataState[key] = data[key]
                }

            })
            .catch(fail)
    }
}

/**
 * @param {any} reason
 */
function fail(reason) {
    console.error("Submit request failed.", reason)
}

/**
 * @param {SubmitEvent} e
 */
export function onsubmit(e) {
    e.preventDefault()
    const headers = {"Accept": "application/json"}
    /** @type {HTMLFormElement} */
    const element = e.target
    const form = new FormData(element)
    const method = element.method

    if (method === "get" || method === "GET") {
        const data = new URLSearchParams();
        for (const [key, value] of form) {
            data.append(key, value.toString());
        }
        const search = data.toString()
        const queryString = `?${search}`
        fetch(queryString, {method, headers}).then(done(queryString)).catch(fail)
        return
    }

    fetch("?", {method, headers, body: form}).then(done(""))
}