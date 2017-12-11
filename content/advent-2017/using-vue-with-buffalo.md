+++
title = "Using Vue.js with Buffalo"
linktitle = "Using Vue.js with Buffalo"
author = ["Mark Bates"]
date = "2017-12-16T00:00:00"
series = ["Advent 2017"]

+++

When I'm writing web applications, I don't tend use the latest JavaScript front-end hotness, instead I prefer to reload the entire page on a request, or at the most, a small section of it. Today, however, many developers love to write JSON back ends and write their front-end logic using JavaScript.

In this article, we're going to do just that. We're going to create a small [Buffalo](https://gobuffalo.io) application that speaks JSON, and we'll create a small front-end to talk to that using [Vue.js](https://vuejs.org). All I ask is your understanding that I'm not a front-end developer, so I'm sure there are plenty of improvements that could be made to my Vue code. Be gentle.

> **NOTE:** To follow along with this article you will need Buffalo installed along with Node/NPM for the front end assets.


## The Go Side

The first step on our journey to front-end greatness starts with creating a new Buffalo application. Since Buffalo is Go code, we have to create our project **inside** our `GOPATH`.

```bash
$ cd $GOPATH/src/github.com/<username>
$ buffalo new vuer
$ cd vuer
$ buffalo db create -a
```

When generating the new Buffalo application we could have used the `--api`, which sets the application up to be a JSON API, and doesn't install any of the front-end services such as templates, JavaScript, CSS, etc... however, since we need those for this particular application we decided to generated a "full" application, instead of a bare bones API application.

We also, using the `buffalo db create -a` command, setup all of the databases for this application as defined in the application's `database.yml`.

> **NOTE:** If you are following along at home, you might need to change the settings inside of the `database.yml` to suit your needs.

### Generating Resources

Next, let us create some resources for application to talk to. This application will hold information about bands and their members.

```bash
$ buffalo generate resource band name bio:text --type=json
$ buffalo generate resource member name instrument band_id:uuid --type=json
$ buffalo db migrate
```

By default when generating resources in Buffalo they are generated as HTML resources, but since this application wants to speak JSON we can use the `--type=json` flag to tell the resource generator to use JSON responses, and not create any HTML templates.

Inside of the `actions/app.go` we should now have two lines that look like this:

```go
// actions/app.go

app.Resource("/bands", BandsResource{&buffalo.BaseResource{}})
app.Resource("/members", MembersResource{&buffalo.BaseResource{}})
```

Let's create a new `api` group and hang these resources off of that. To do that we can create a new group, `api`, and hang the `BandsResource` off of it. We're going to also want to nest the `MembersResource` under the `band` group as well.

```go
// actions/app.go

api := app.Group("/api")
band := api.Resource("/bands", BandsResource{&buffalo.BaseResource{}})
band.Resource("/members", MembersResource{&buffalo.BaseResource{}})
```

With these changes in place if we were to print off our table, `buffalo task routes`, it would look something similar to this:

```text
METHOD | PATH
------ | ----
get    | /api/bands
post   | /api/bands
get    | /api/bands/new
get    | /api/bands/{band_id}
put    | /api/bands/{band_id}
delete | /api/bands/{band_id}
get    | /api/bands/{band_id}/edit
get    | /api/bands/{band_id}/members
post   | /api/bands/{band_id}/members
get    | /api/bands/{band_id}/members/new
get    | /api/bands/{band_id}/members/{member_id}
put    | /api/bands/{band_id}/members/{member_id}
delete | /api/bands/{band_id}/members/{member_id}
get    | /api/bands/{band_id}/members/{member_id}/edit
```

Now that we have generated and mapped all of our resources we need to tweak the `MembersResource` so that it is scoped to the requested band. We don't want to show members of the Rolling Stones if someone is requested the members of the Beatles.

For example, in `MembersResource#List` we would change the call that finds all of the members to scope it to the `band_id` on the request.

```go
// actions/members.go

// before
if err := q.All(members); err != nil {
  return errors.WithStack(err)
}

// after
if err := q.Where("band_id = ?", c.Param("band_id")).All(members); err != nil {
  return errors.WithStack(err)
}
```

After making these changes in the `MembersResource#List`, `MembersResource#Create`, `MembersResource#Update`, and `MembersResource#Destroy` actions we are almost finished with the Go side of the application.

The final step, before we can move the JavaScript side is to set a catch-all route. A catch-all route will allow for the application to accept any URL we haven't already defined and let the Vue router handle those requests instead.

In the `actions/app.go` file we can add this catch-all route right before the route mapping `/` to the `HomeHandler`.

```go
// actions/app.go

app.GET("/{path:.+}", HomeHandler)
```

With that we are finished with the Go side of the application. Now we can turn our attention to hooking up Vue.js.

## The JavaScript Side

To get started on the JavaScript side we first need to install four Node modules, to do this we will use [Yarn](https://yarnpkg.com/). These modules will allow us access to Vue, a router for Vue, and a few other pieces like the ability to compile Vue templates.

```bash
$ yarn add vue vue-loader vue-router vue-template-compiler
```

With the proper modules installed we need to tell Webpack how to work with these modules. To do that we need to add the following entry to the `webpack.config.js` file that Buffalo generates.

```javascript
// webpack.config.js

// ...
modules.exports = {
  resolve: {
    alias: {
      vue$: `${__dirname}/node_modules/vue/dist/vue.esm.js`,
      router$: `${__dirname}/node_modules/vue-router/dist/vue-router.esm.js`
    }
  },
  // ...
}
// ...
```


With all the glue in place, and the proper modules installed, we can write our Vue application. Since this article isn't about learning Vue, and since I'm not a Vue expert, I'm going to simply show you the code I wrote to make my simple application work.

```javascript
// assets/js/application.js

require("expose-loader?$!expose-loader?jQuery!jquery");
require("bootstrap-sass/assets/javascripts/bootstrap.js");

import Vue from "vue";
import VueRouter from "router";
Vue.use(VueRouter);

import BandComponent from "./components/band.vue";
import MembersComponent from "./components/members.vue";

const routes = [
  {path: "/band/:id", component: MembersComponent, name: "showBand"},
  {path: "/", component: BandComponent}
];

const router = new VueRouter({
  mode: "history",
  routes
});

const app = new Vue({
  router
}).$mount("#app");
```

```vue
// assets/js/components/band.vue

<template>
<div>
  <h1 class="page-header">Bands</h1>

  <ul class="list-unstyled">
    <li v-for="band in bands">
      <router-link :to='{name: "showBand", params: {id: band.id}}'>
        <h2>
          {{ band.name }}
        </h2>
      </router-link>
    </li>
  </ul>
</div>
</template>

<script charset="utf-8">
export default {
  data() {
    return {
      bands: []
    };
  },

  created() {
    this.fetchData();
  },

  watch: {
    $route: "fetchData"
  },

  methods: {
    fetchData: function() {
      let req = $.getJSON("/api/bands");
      req.done(data => {
        this.bands = data;
      });
    }
  }
};
</script>
```

```vue
// assets/js/components/members.vue

<template>
<div>
  <h1 class="page-header">{{band.name}}</h1>

  <blockquote>
    {{band.bio}}
  </blockquote>

  <ul class="list-unstyled">
    <li v-for="member in members">
      <h2>
        {{member.name}} - {{member.instrument}}
      </h2>
    </li>
  </ul>

</div>
</template>

<script charset="utf-8">
export default {
  data() {
    return {
      band: {},
      members: {}
    };
  },

  created() {
    this.fetchData();
  },

  watch: {
    $route: "fetchData"
  },

  methods: {
    fetchData: function() {
      let id = this.$route.params.id;

      let req = $.getJSON(`/api/bands/${id}`);
      req.done(data => {
        this.band = data;
      });

      req = $.getJSON(`/api/bands/${id}/members`);
      req.done(data => {
        this.members = data;
      });
    }
  }
};
</script>
```

In order to get `*.vue` files to work with Webpack, we need to add a rule that tells Webpack to use the `vue-loader` plugin to process those files for us. We can update the `webpack.config.js` file and add a rule to that affect.

```javascript
// webpack.config.js

// ...
modules.exports = {
  // ...
  module: {
    rules: [
      // ...
      {
        test: /\.vue/,
        loader: "vue-loader"
      },
      // ...
    ]
    // ...
  }
  // ...
}
```

## Putting It All together

With all of that in place we are almost ready to start our application and try it out. We just need one more bit of glue code, and a script to seed the database with a few bands to start with.

Let's start with the glue code. In order to start the Vue application we need to give it an HTML element to bind to. To do this we can replace the contents of `templates/index.html` with the following.

```html
// templates/index.html

<div id="app">
  <router-link to="/">Home</router-link>
  <router-view></router-view>
</div>
```

The above code will not only allow Vue to bind to the page, but it also provides an element for the Vue router to attach to and replace the content of as we navigate pages.

Finally, let's add a script to seed the database with a few bands. When we generated the application, Buffalo, created a new file, `grifts/db.go`. This file contains a `db:seed` task. The purpose of this task is to let us write a seed script for our database.

We can replace the placeholder of this script with the following:

```go
var _ = grift.Namespace("db", func() {

  grift.Desc("seed", "Seeds a database")
  grift.Add("seed", func(c *grift.Context) error {
    if err := models.DB.TruncateAll(); err != nil {
      return errors.WithStack(err)
    }

    band := &models.Band{
      Name: "The Beatles",
      Bio:  "4 fun loving lads from Liverpool.",
    }
    if err := models.DB.Create(band); err != nil {
      return errors.WithStack(err)
    }
    members := models.Members{
      {Name: "John Lennon", Instrument: "Guitar"},
      {Name: "Paul McCartney", Instrument: "Bass"},
      {Name: "George Harrison", Instrument: "Guitar"},
      {Name: "Ringo Starr", Instrument: "Drums"},
    }
    for _, m := range members {
      m.BandID = band.ID
      if err := models.DB.Create(&m); err != nil {
        return errors.WithStack(err)
      }
    }

    band = &models.Band{
      Name: "The Monkees",
      Bio:  "4 fun loving lads assembled by a TV studio",
    }
    if err := models.DB.Create(band); err != nil {
      return errors.WithStack(err)
    }
    members = models.Members{
      {Name: "Mike Nesmith", Instrument: "Guitar"},
      {Name: "Davy Jones", Instrument: "Voice"},
      {Name: "Peter Tork", Instrument: "Guitar"},
      {Name: "Mikey Dolenz", Instrument: "Drums"},
    }
    for _, m := range members {
      m.BandID = band.ID
      if err := models.DB.Create(&m); err != nil {
        return errors.WithStack(err)
      }
    }

    return nil
  })

})
```

This script can be run like such:

```bash
$ buffalo task db:seed
```

Buffalo uses the [grift](https://gobuffalo.io/docs/tasks) task runner for these types of simple, repeatable scripts.

With seed data in place we can launch the application in development mode.

```bash
$ buffalo dev
```

The `buffalo dev` command will not only start the application at `http://localhost:3000`, it also watches your Go files, and assets, for changes. If there are changes then Buffalo will recompile your application, and/or assets, and restart it.

## Conclusion

In this article we built a brand new, database backed, Buffalo application that speaks JSON. We also quickly built a single page application on top of the Buffalo application, thanks to Buffalo's Webpack asset pipeline.

Hopefully this article has inspired you to try your favorite framework on top of Buffalo. Perhaps using [GopherJS](https://github.com/gopherjs/gopherjs). :)

Full source code can be found at [https://github.com/gobuffalo/vuerecipe](https://github.com/gobuffalo/vuerecipe).

---

## About the Author

Mark is the co-founder of [PaperCall.io](https://papercall.io), a platform for connecting technical events with high quality content and speakers. Mark is also a partner at [Gopher Guides](http://bit.ly/2Bcnw7C), the industry leader for Go training and conferences. In his spare time Mark leads development of the Go web framework [Buffalo](https://gobuffalo.io).
