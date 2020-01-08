// Copyright © 2015 Martin Bertschler <mbertschler@gmail.com>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/mbertschler/crowd"
)

var (
	port      string
	path      string
	userStore stringStore
)

type stringStore struct {
	*crowd.Store
}

func (s stringStore) CookieGetData(w http.ResponseWriter, r *http.Request) (*crowd.User, string, error) {
	u, err := s.CookieGet(w, r)
	data, ok := u.Data.(string)
	if !ok {
		data = "&nbsp;"
	}
	return u, data, err
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.StringVar(&port, "port", ":8001", "Port for http server")
	flag.Parse()

	userStore = stringStore{crowd.NewMemoryStore()}

	http.HandleFunc("/", index)
	http.HandleFunc("/login", login)
	http.HandleFunc("/register", register)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/delete", del)
	http.HandleFunc("/rename", rename)
	http.HandleFunc("/password", password)
	http.HandleFunc("/save", save)

	log.Println("Testapp for \"github.com/mbertschler/crowd\"")
	log.Println("Serving HTTP at http://localhost" + port)
	if path != "" {
		log.Println("Saving crowd DB at " + path)
	}
	log.Println("------------------------------------------")
	log.Fatal(http.ListenAndServe(port, nil))
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}
	_, err := userStore.CookieLogin(w, r,
		r.PostFormValue("user"),
		r.PostFormValue("pass"),
	)
	if err != nil {
		log.Println("Login error:", err)
		w.Write(errorPage(fmt.Sprintln("Login error:", err)))
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
func register(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}
	_, err := userStore.CookieRegister(w, r,
		r.PostFormValue("user"),
		r.PostFormValue("pass"))
	if err != nil {
		log.Println("Register error:", err)
		w.Write(errorPage(fmt.Sprintln("Register error:", err)))
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
func logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}
	_, err := userStore.CookieLogout(w, r)
	if err != nil {
		log.Println("Logout error:", err)
		w.Write(errorPage(fmt.Sprintln("Logout error:", err)))
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
func del(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}
	_, err := userStore.CookieDelete(w, r)
	if err != nil {
		log.Println("Delete error:", err)
		w.Write(errorPage(fmt.Sprintln("Delete error:", err)))
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
func rename(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}
	_, err := userStore.CookieSetUsername(w, r, r.PostFormValue("name"))
	if err != nil {
		log.Println("Rename error:", err)
		w.Write(errorPage(fmt.Sprintln("Rename error:", err)))
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
func password(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}
	_, err := userStore.CookieSetPassword(w, r, r.PostFormValue("pass"))
	if err != nil {
		log.Println("Password error:", err)
		w.Write(errorPage(fmt.Sprintln("Password error:", err)))
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
func save(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}
	_, err := userStore.CookieSaveData(w, r, r.PostFormValue("val"))
	if err != nil {
		log.Println("Save error:", err)
		w.Write(errorPage(fmt.Sprintln("Save error:", err)))
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
func index(w http.ResponseWriter, r *http.Request) {
	user, data, err := userStore.CookieGetData(w, r)
	if err != nil {
		log.Println("Index error:", err)
		w.Write(errorPage(fmt.Sprintln("Index error:", err)))
		return
	}

	w.Write([]byte(header + `
		<h1>Testapp for package <a href="https://github.com/mbertschler/crowd">"github.com/mbertschler/crowd"</a></h1>
		<table border="1">
			<thead>
					<th>Variable</th>
					<th>Value</th>
			</thead>
			<tbody>
				<tr>
					<td>Session ID</td>
					<td style="word-break: break-all;">` + fmt.Sprint(user.Session.ID) + `</td>
				</tr>
				<tr>
					<td>Session Expires</td>
					<td>` + fmt.Sprint(user.Session.Expires.Format("2006 Jan 02 15:04:05 MST -0700")) + `</td>
				</tr>
				<tr>
					<td>Session LastCon</td>
					<td>` + fmt.Sprint(user.Session.LastAccess.Format("2006 Jan 02 15:04:05 MST -0700")) + `</td>
				</tr>
				<tr>
					<td>Session LoggedIn</td>
					<td>` + fmt.Sprint(user.LoggedIn) + `</td>
				</tr>
				<tr>
					<td>Session UserID</td>
					<td>` + fmt.Sprint(user.Session.UserID) + `</td>
				</tr>
				<tr>
					<td>Username</td>
					<td>` + fmt.Sprint(user.Name) + `</td>
				</tr>
				<tr>
					<td>Data</td>
					<td>` + data + `</td>
				</tr>
			</tbody>
		</table>
		<div>
			<h2>Register</h2>
			<form action="/register" method="POST">
				<input type="text" name="user" placeholder="Username"/> <br/>
				<input type="password" name="pass" placeholder="Password"/> <br/>
				<button type="submit">Register</button>
			</form>
		</div>
		<div>
			<h2>Login</h2>
			<form action="/login" method="POST">
				<input type="text" name="user" placeholder="Username"/> <br/>
				<input type="password" name="pass" placeholder="Password"/> <br/>
				<button type="submit">Login</button>
			</form>
		</div>
		<div>
			<h2>Logout</h2>
			<form action="/logout" method="POST">
				<button type="submit">Logout</button>
			</form>
		</div>
		<div>
			<h2>Delete User</h2>
			<form action="/delete" method="POST">
				<button type="submit">Delete</button>
			</form>
		</div>
		<br/>
		<div>
			<h2>Change Username</h2>
			<form action="/rename" method="POST">
				<input type="text" name="name" placeholder="New username"/> <br/>
				<button type="submit">Change Username</button>
			</form>
		</div>
		<div>
			<h2>Change Password</h2>
			<form action="/password" method="POST">
				<input type="text" name="pass" placeholder="New password"/> <br/>
				<button type="submit">Change Password</button>
			</form>
		</div>
		<div>
			<h2>Set Data</h2>
			<p>` + data + `</p>
			<form action="/save" method="POST">
				<input type="text" name="val" placeholder="Value"/> <br/>
				<button type="submit">Save</button>
			</form>
		</div>` + footer))
}

func errorPage(in string) []byte {
	return []byte(header + `
		<h1>Testapp for package <a href="https://github.com/mbertschler/crowd">"github.com/mbertschler/crowd"</a></h1>
		<h2>Error</h2>
		<p>` + in + `</p>
		<a href="/"><button type="submit">Back</button></a>
	` + footer)
}

var header = `
<html>
	<head>
		<style>
			body{
				font-family: sans-serif;
				font-size:16px;
				color: #333;
			}
			td {
			    padding: 4px;
			}
			th {
			    padding: 6px;
			    font-size:19px;
			    background-color:#ddd;
			}
			input, button {
				font-size:16px;
				padding:3px;
				margin-top:6px;
			}
			div {
				display: inline-block;
				width: 200px;
				vertical-align:top;
			}
		</style>
	</head>
	<body>`

var footer = `
	</body>
</html>
`
