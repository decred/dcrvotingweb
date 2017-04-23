module.exports = function(grunt) {
    grunt.initConfig({
        copy: {
            main: {
                files: [{
                        expand: true,
                        cwd: 'src/',
                        src: ['*'],
                        dest: '../public/',
                        filter: 'isFile'
                    },
                    {   
                        expand: true,
                        cwd: 'src/css',
                        src: ['**/*'],
                        dest: '../public/css',
                        filter: 'isFile'
                    },
                    {   
                        expand: true,
                        cwd: 'src/images',
                        src: ['**/*'],
                        dest: '../public/images',
                        filter: 'isFile'
                    },
                    {   
                        expand: true,
                        cwd: 'src/js',
                        src: ['**/*'],
                        dest: '../public/js',
                        filter: 'isFile'
                    },
                    {   
                        expand: true,
                        cwd: 'src/fonts',
                        src: ['**/*'],
                        dest: '../public/fonts',
                        filter: 'isFile'
                    },
                ]

            }
        },

        concat: {
            dist: {
                src: [
                    'bower_components/jquery/dist/jquery.min.js',
                    'src/js/chart.min.js',
                    'src/js/chart-extensions.js',
                    'src/js/decred-hardforkwebsite.js',
                    'src/js/hfsite.js'
                ],
                dest: '../public/js/complete.js'
            }
        },

        uglify: {
            my_target: {
                files: {
                    '../public/js/complete.js': ['../public/js/complete.js']
                }
            }
        },
        watch: {

            html: {
                files: ['src/*'],
                tasks: ['copy'],
            },

            img: {
                files: ['src/images/*.{png,jpg,gif}'],
                tasks: ['imagemin'],
                options: {
                    livereload: true
                }
            },
        }
    });



    grunt.loadNpmTasks('grunt-contrib-copy');
    grunt.loadNpmTasks('grunt-contrib-uglify');
    grunt.loadNpmTasks('grunt-contrib-concat');
    grunt.loadNpmTasks('grunt-contrib-watch');

    grunt.registerTask('default', ['copy', 'concat', 'uglify', ]);
};
