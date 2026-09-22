let API_URL = "";
let USER_POOL_ID = "";
let CLIENT_ID = "";
let COGNITO_ENDPOINT = "";

let idToken = "";

async function loadConfig() {
    try {
        const res = await fetch("config.json");
        if (!res.ok) throw new Error(`config.json HTTP ${res.status}`);
        const cfg = await res.json();
        API_URL = cfg.API_URL || "";
        USER_POOL_ID = cfg.USER_POOL_ID || "";
        CLIENT_ID = cfg.CLIENT_ID || "";
        COGNITO_ENDPOINT = cfg.COGNITO_ENDPOINT || cfg.AWS_ENDPOINT_URL || "http://localhost.floci.io:4566";
        if (!API_URL) throw new Error("API_URL missing in config.json");
    } catch (err) {
        document.getElementById("posts-container").innerText = "Failed to load config.json.";
        throw err;
    }
}

async function fetchPosts() {
    try {
        const res = await fetch(API_URL);
        const data = await res.json();
        const container = document.getElementById("posts-container");
        container.innerHTML = "";
        if (!data || data.length === 0) {
            container.innerHTML = "<p>No blog posts found.</p>";
            return;
        }
        data.forEach(post => {
            const id = post.postId || post.id || "";
            container.innerHTML += `
                <div class="card">
                    <h3>${post.title}</h3>
                    <p>${post.content}</p>
                    <small>Posted on: ${new Date(post.createdAt).toLocaleString()}</small><br>
                    ${id ? `<button onclick="fetchPostById('${id}')">View Details</button>` : ""}
                </div>`;
        });
    } catch (err) {
        document.getElementById("posts-container").innerText = "Failed to load posts.";
    }
}

async function fetchPostById(postId) {
    try {
        const res = await fetch(`${API_URL}/${postId}`);
        const container = document.getElementById("posts-container");
        if (res.status === 404) {
            container.innerHTML = `<p>Post not found.</p><button onclick="fetchPosts()">Back to all posts</button>`;
            return null;
        }
        if (!res.ok) {
            throw new Error(`HTTP ${res.status}`);
        }
        const post = await res.json();
        container.innerHTML = `
            <div class="card">
                <h3>${post.title}</h3>
                <p>${post.content}</p>
                <small>Posted on: ${new Date(post.createdAt).toLocaleString()}</small><br>
                <small>ID: ${post.postId || postId}</small><br><br>
                <button onclick="fetchPosts()">Back to all posts</button>
            </div>`;
        return post;
    } catch (err) {
        document.getElementById("posts-container").innerHTML = `<p>Failed to load post.</p><button onclick="fetchPosts()">Back to all posts</button>`;
        return null;
    }
}

function login() {
    const email = document.getElementById("email").value;
    const password = document.getElementById("password").value;
    document.getElementById("auth-status").innerText = "Logging in to " + COGNITO_ENDPOINT + "...";

    const poolData = { UserPoolId: USER_POOL_ID, ClientId: CLIENT_ID, endpoint: COGNITO_ENDPOINT };
    const userPool = new AmazonCognitoIdentity.CognitoUserPool(poolData);
    // Ensure underlying provider also hits floci, not real AWS
    if (userPool.client && COGNITO_ENDPOINT) {
        try { userPool.client.endpoint = COGNITO_ENDPOINT; } catch (e) {}
    }
    const authenticationData = { Username: email, Password: password };
    const authenticationDetails = new AmazonCognitoIdentity.AuthenticationDetails(authenticationData);
    const userData = { Username: email, Pool: userPool };
    const cognitoUser = new AmazonCognitoIdentity.CognitoUser(userData);

    cognitoUser.authenticateUser(authenticationDetails, {
        onSuccess: function (result) {
            idToken = result.getIdToken().getJwtToken();
            document.getElementById("auth-status").innerText = "Login successful! (floci)";
            document.getElementById("create-post-sec").classList.remove("hidden");
        },
        onFailure: function (err) {
            document.getElementById("auth-status").innerText = "Error: " + (err.message || JSON.stringify(err));
        }
    });
}

async function createPost() {
    const title = document.getElementById("title").value;
    const content = document.getElementById("content").value;

    const res = await fetch(API_URL, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
            "Authorization": idToken
        },
        body: JSON.stringify({ title, content })
    });

    if (res.ok) {
        alert("Post published successfully!");
        document.getElementById("title").value = "";
        document.getElementById("content").value = "";
        fetchPosts();
    } else {
        alert("Unauthorized or error publishing post.");
    }
}

loadConfig().then(fetchPosts).catch(console.error);
