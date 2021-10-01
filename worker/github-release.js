// utils
function versionCompare(v1, v2, options) {
    // https://stackoverflow.com/questions/6832596/how-to-compare-software-version-number-using-js-only-number/53387532
    var lexicographical = options && options.lexicographical,
        zeroExtend = options && options.zeroExtend,
        v1parts = v1.split('.'),
        v2parts = v2.split('.');

    function isValidPart(x) {
        return (lexicographical ? /^\d+[A-Za-z]*$/ : /^\d+$/).test(x);
    }

    if (!v1parts.every(isValidPart) || !v2parts.every(isValidPart)) {
        return NaN;
    }

    if (zeroExtend) {
        while (v1parts.length < v2parts.length) v1parts.push("0");
        while (v2parts.length < v1parts.length) v2parts.push("0");
    }

    if (!lexicographical) {
        v1parts = v1parts.map(Number);
        v2parts = v2parts.map(Number);
    }

    for (var i = 0; i < v1parts.length; ++i) {
        if (v2parts.length === i) {
            return 1;
        }

        if (v1parts[i] === v2parts[i]) {

        } else if (v1parts[i] > v2parts[i]) {
            return 1;
        } else {
            return -1;
        }
    }

    if (v1parts.length !== v2parts.length) {
        return -1;
    }

    return 0;
}

async function getGithubReleasesJson(event) {
    const cacheUrl = new URL(event.request.url)
    const cacheKey = new Request(cacheUrl.toString(), event.request)
    const cache = caches.default
    let response = await cache.match(cacheKey)
    if (!response) {
        console.log("start fetch");
        response = await fetch(`https://api.github.com/repos/maxisme/notifi/releases`, {
            headers: {
                'User-Agent': event.request.headers.get('user-agent')
            }
        })
        console.log("end fetch");
        if (response.status !== 200) {
            throw 'cannot fetch release: ' + response.statusText;
        }
        response = new Response(response.body, response)
        response.headers.append("Cache-Control", "s-maxage=120") // cache for 2 mins
        event.waitUntil(cache.put(cacheKey, response.clone()))
    }
    return response.json()
}

async function getGithubRelease(event) {
    const {searchParams} = new URL(event.request.url)
    const isDevelop = searchParams.get('develop') != null
    const version = searchParams.get("version");

    const results = await getGithubReleasesJson(event)

    let release = null;
    for (const result of results) {
        if (!result['draft']) {
            if (isDevelop && result['prerelease']) {
                release = result;
                break;
            } else if (!isDevelop && !result['prerelease']) {
                release = result;
                break;
            }
        }
    }

    if (version != null) {
        if (versionCompare(version, release['tag_name']) === -1) {
            // there is a newer version
            return new Response("", {status: 200})
        }
        return new Response("", {status: 404})
    }

    // find .dmg download file
    let dmg_download_url = "";
    for (const asset of release['assets']) {
        if (asset['browser_download_url'].indexOf(".dmg") !== -1) {
            dmg_download_url = asset['browser_download_url'];
        }
    }

    const xml = `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0" xmlns:sparkle="https://notifi.it/xml-namespaces/sparkle" xmlns:dc="https://notifi.it/dc/elements/1.1/">
  <channel>
    <item>
        <title>${release['name']}</title>
        <description><![CDATA[
            ${release['body']}
        ]]>
        </description>
        <pubDate>${release['published_at']}</pubDate>
        <enclosure url="${dmg_download_url}" sparkle:version="${release['tag_name']}"/>
    </item>
  </channel>
</rss>`;
    return new Response(xml, {
        status: 200, headers: {
            'Content-Type': 'application/xml'
        }
    })
}

////////////
// worker //
////////////

addEventListener("fetch", (event) => {
    event.respondWith(
        getGithubRelease(event).catch(
            (err) => new Response(err.stack, {status: 500})
        )
    );
});
