+++
author = ["Jesus R. Gonzalez L."]
date = "2019-12-06T10:07:34+09:00"
title = "Como construir sitios web con HUGO"
series = ["Tutorial"]
+++

Esta publicación tiene como objetivo enseñar a como construir sitios web utilizando HUGO y como subirlo a un hosting.

Hugo es un generador de sitios web estáticos construidos con el lenguaje GO, fue creado en el año 2013. Con él se pueden generar sitios web en poco tiempo, versátiles y muy rapidos de hecho su lema es: "The world’s fastest framework for building websites".  Algunos casos notables en donde se construyeron sitios web con HUGO son:

- Kubernetes        (<https://kubernetes.io/>)
- Netlify           (<https://www.netlify.com/>)
- Smashingmagazine  (<https://www.smashingmagazine.com/>)
- Cloudflare        (<https://developers.cloudflare.com/>)
- Litecoin          (<https://litecoin.org/>)

Para trabajar con Hugo necesitamos instalar los siguientes requerimientos:

1. __Git__ (Software para control de versiones)
2. __GO__ (Lenguaje de Programación)
3. __Chocolatey__ (Manejador de paquetes de Windows. Utilice Windows 7)

HUGO también tiene instrucciones para instalarse en otros sistemas operativos, para más información visite este link:
<https://gohugo.io/getting-started/installing/#quick-install>

### 1. Instalación de Git

Primeramente hay que descargarlo en el siguiente link:

<https://git-scm.com/downloads>.

Después de haber sido instalado abrimos __Git Bash__, para comprobar que nuestra instalación fue exitosa y no tuvo errores, ingresamos el siguiente comando que muestra la versión de git instalada.

    git --version

### 2. Instalación de GO (Golang)

Se descarga de la página oficial:

<https://golang.org/dl/>

Luego de haber sido descargado pasamos al proceso de instalación. Cuando finalice su instalación hay agregar una variable en __Enviroment Variables__ –> __User Variables__ con el nombre de __GOPATH__  y en la sección __value__ la dirección de la carpeta donde se instalo.
Ya con eso se podrá desde la consola ingresar los comandos de GO. Ingresamos el siguiente comando que debería mostrar la versión instalada:

    go version

### 3. Instalación de Chocolatey

Procedemos a abrir __powershell.exe__ como administrador, de lo contrario no se instalara.

Ahora toca ingresar el siguiente comando en la consola:

    Set-ExecutionPolicy Bypass -Scope Process -Force; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))

Luego de finalizar la instalación ingresamos el comando que nos mostrara la versión de Chocolatey:

    choco  /?

### Instalación de HUGO

Lo primero que debemos hacer es dirigirnos a la carpeta donde se va a instalar HUGO, abrimos la consola __Git Bash__ en modo administrador e ingresamos el siguiente comando:

    choco install hugo

Finalizado el proceso, podremos ver la versión de HUGO que se instalo:

    hugo version

### a. Crear un sitio web

 Se crea con el comando:

    hugo new site nombre_tu_preferencia

Ejemplo:

    hugo new site gopherblog

Nos dirigimos a la carpeta del nuevo sitio web:

    cd gopherblog

Iniciamos Git en el proyecto creado:

    git init

### b. Añadir Tema

Utilice el tema: __hugo-sustain__, este es el link del repositorio del tema que elegi:

<https://github.com/nurlansu/hugo-sustain>

Nota: Hugo posee distintos temas para distintos usos. Elige el que se adapta a tus necesidades:
<https://themes.gohugo.io/>

Ahora toca clonar su repositorio y guardarlo en la carpeta __themes__ de su proyecto.

    git clone https://github.com/nurlansu/hugo-sustain.git themes/hugo-sustain

### c. Modificación del Tema

En el archivo __config.toml__ de su proyecto se puede modificar variables que son usadas en el tema.

Debemos empezar por agregar el nombre del tema que se eligió:

    theme = 'hugo-sustain'

El formato de los permalink:

    post = "/:year/:month/:day/:slug" 

Las variables __params__ son donde podemos cambiar algunos aspectos de la página inicial del tema como el autor del proyecto, la descripción y el avatar. En la carpeta __static__, hay que crear una carpeta llamada __img__, se guardara una imagen que puede ser en formato png o jpg. El creador del tema que decidi utilizar recomienda que la imagen sea de una dimensión de 190 x 190 pixeles.

    [params]
    avatar = "profile.jpg"
    author = "Jesus Gonzalez" 
    description = "Describe your website"

Las variables __params.social__ son las encargadas de capturar el nombre de usuario de las distintas redes sociales.

    [params.social]
    Github        = "username"
    Email         = "email@example.com"
    Twitter       = "username"
    LinkedIn      = "username"
    Stackoverflow = "username"
    Medium        = "username"
    Telegram      = "username"

Con la variable __menu.main__ se puede agregar secciones a su tema. Ejemplo de como crear una sección:

    ## Main Menu
    [[menu.main]]
    name = "blog"
    weight = 100
    identifier = "blog"
    url = "/blog/"

El archivo final debería tener el siguiente formato:

    baseURL = 'http://example.org/'
    languageCode = "en-us"
    title = "gopherblog"
    theme = "hugo-sustain"

    [permalinks]
    post = "/:year/:month/:day/:slug"

    [params]
    avatar = "profile.jpg"
    author = "Jesus Gonzalez"
    description = "tutorial como construir con HUGO"

    [params.social]
    Github        = "username"
    Email         = "email@example.com"
    Twitter       = "username"
    LinkedIn      = "username"
    Stackoverflow = "username"
    Medium        = "username"
    Telegram      = "username"

    ## Main Menu
    [[menu.main]]
    name = "blog"
    weight = 100
    identifier = "blog"
    url = "/blog/"

### d. Creación del primer Post

Aprovechando los beneficios de HUGO se pueden crear posts de forma automatica ingresando un comando que genera un archivo donde tu podras ingresar el contenido a publicar. El archivo es en formato __markdown__

    hugo new blog/primer_post.md

### d. Inicio del servidor de HUGO

Para iniciar el servidor local se ingresa:

    hugo server –D

Otro comando es:

    hugo serve 

Este comando va a funcionar, pero los post creados no se van a mostrar porque se encuentran en estado: DRAFT = true, para que se visualicen es necesario eliminar una línea de los posts.

    draft: true

### e. Subir el proyecto a un Hosting

Para este tutorial se utilizo __Github Pages__.

#### Pasos para subir el proyecto

1. Un repositorio con el nombre de su proyecto. Ejemplo:

    > hugo_blog

2. Un segundo repositorio con el nombre de usuario de su cuenta de github

    > username.github.io

    (Recomendación: Este repositorio debe ser creado con su Readme.md).

3. En la consola __git bash__ que se encuentra abierta en la carpeta de su proyecto agregamos el siguiente comando:

        git remote add origin git@github.com:username/hugo_blog.git

4. Ahora debes agregar sus archivos al área staging de git eso se realiza con los comandos:

        git add .
        git commit –m “first commit”

5. Luego debes subir al repositorio con el nombre del proyecto.

        git push –u origin master

6. Hay que clonar el otro repositorio.

        git submodule add -b master git@github.com:username/username.github.io.git public

7. Debes generar la carpeta __public__ donde se almacenaran los archivos que son necesarios para el hosting

        hugo

8. Para finalizar es necesario crear un Shell script con el nombre de __deploy.sh__ que debe tener lo siguiente:

        #!/bin/sh

        # If a command fails then the deploy stops
        set -e
            
        printf "\033[0;32mDeploying updates to GitHub...\033[0m\n"

        # Build the project.
        hugo -t hugo-sustain # if using a theme, replace with `hugo -t <YOURTHEME>`

        # Go To Public folder
        cd public

        # Add changes to git.
        git add .

        # Commit changes.
        msg="rebuilding site $(date)"
            if [ -n "$*" ]; then
                msg="$*"
            fi
        git commit -m "$msg"

        # Push source and build repos.
        git push origin master

9. Antes de ejecutar el script debes cambiar en el archivo __config.toml__ el valor de la variable __baseURL__ con:

    <https://username.github.io/>  

10. Ya cambiada esa línea puedes poner en ejecución el script.

    sh deploy.sh

Ahora su sitio debería estar disponible.

<https://username.github.io>

### Conclusión

El tutorial fue creado para que las personas que nuncan han trabajado __HUGO__ puedan desarrollar su sitio web en poco tiempo, sin tantas complicaciones y sepan como subirlo a un hosting.

Déjame saber que te pareció el artículo en mi twitter:

[Twitter](https://twitter.com/gonzalezlrjesus)

[Portafolio](https://gonzalezlrjesus.github.io/) (Mi Portafolio fue creado con __HUGO__ y con el tema __hugo-sustain__)
