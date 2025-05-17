/**
 * LessonCraft Lesson Editor
 * 
 * This file contains the JavaScript functionality for the lesson editor,
 * including real-time preview, validation, and save/load capabilities.
 */

(function() {
  'use strict';

  // Register the lesson editor controller
  angular.module('DockerPlay').controller('LessonEditorController', [
    '$scope', '$mdDialog', '$http', '$sce', '$timeout', 'LessonService',
    function($scope, $mdDialog, $http, $sce, $timeout, LessonService) {
      // Initialize the lesson object
      $scope.lesson = {
        title: '',
        description: '',
        steps: []
      };

      // Initialize the markdown content
      $scope.markdownContent = '';
      $scope.previewSteps = [];
      $scope.validationErrors = [];
      $scope.isEditing = false;
      $scope.editingLessonId = null;
      $scope.isDraft = false;

      // Initialize with a lesson if editing
      $scope.initWithLesson = function(lessonId) {
        if (!lessonId) return;
        
        $scope.isEditing = true;
        $scope.editingLessonId = lessonId;
        
        LessonService.getLesson(lessonId).then(function(lesson) {
          $scope.lesson = lesson;
          $scope.isDraft = lesson.isDraft || false;
          
          // Convert lesson to markdown
          $scope.markdownContent = lessonToMarkdown(lesson);
          $scope.updatePreview();
        });
      };

      // Convert a lesson object to markdown
      function lessonToMarkdown(lesson) {
        let markdown = `# ${lesson.title}\n\n${lesson.description}\n\n`;
        
        lesson.steps.forEach(function(step, index) {
          markdown += `## Step ${index + 1}\n\n${step.content}\n\n`;
          
          if (step.commands && step.commands.length > 0) {
            markdown += "```docker\n" + step.commands.join('\n') + "\n```\n\n";
          }
          
          if (step.expected) {
            markdown += "```expect\n" + step.expected + "\n```\n\n";
          }
          
          if (step.question) {
            markdown += "```question\n" + step.question + "\n```\n\n";
          }
        });
        
        return markdown;
      }

      // Update the preview when the markdown content changes
      $scope.updatePreview = function() {
        // Parse the markdown content
        parseMarkdown($scope.markdownContent).then(function(lesson) {
          $scope.lesson.title = lesson.title || $scope.lesson.title;
          $scope.lesson.description = lesson.description || $scope.lesson.description;
          $scope.previewSteps = lesson.steps.map(function(step) {
            // Convert markdown content to HTML for preview
            return {
              content: $sce.trustAsHtml(markdownToHtml(step.content)),
              commands: step.commands || [],
              expected: step.expected || '',
              question: step.question || ''
            };
          });
          
          // Validate the lesson
          $scope.validateContent(false);
        });
      };

      // Parse markdown content into a lesson object
      function parseMarkdown(markdown) {
        return $http.post('/api/lessons/parse', { markdown: markdown })
          .then(function(response) {
            return response.data;
          })
          .catch(function(error) {
            console.error('Error parsing markdown:', error);
            return {
              title: '',
              description: '',
              steps: []
            };
          });
      }

      // Simple markdown to HTML conversion for preview
      // This is a basic implementation - in production, use a proper markdown library
      function markdownToHtml(markdown) {
        if (!markdown) return '';
        
        // Convert headers
        let html = markdown.replace(/^# (.*?)$/gm, '<h1>$1</h1>')
                          .replace(/^## (.*?)$/gm, '<h2>$1</h2>')
                          .replace(/^### (.*?)$/gm, '<h3>$1</h3>');
        
        // Convert paragraphs
        html = html.split('\n\n').map(function(p) {
          if (!p.startsWith('<h')) {
            return '<p>' + p + '</p>';
          }
          return p;
        }).join('');
        
        // Convert inline code
        html = html.replace(/`(.*?)`/g, '<code>$1</code>');
        
        // Convert bold
        html = html.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
        
        // Convert italic
        html = html.replace(/\*(.*?)\*/g, '<em>$1</em>');
        
        return html;
      }

      // Insert a code block at the cursor position
      $scope.insertBlock = function(blockType) {
        const textarea = document.querySelector('.markdown-editor textarea');
        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;
        const text = $scope.markdownContent;
        
        let blockTemplate = '';
        switch(blockType) {
          case 'docker':
            blockTemplate = "```docker\n# Enter your commands here\n```\n\n";
            break;
          case 'expect':
            blockTemplate = "```expect\n# Expected output here\n```\n\n";
            break;
          case 'question':
            blockTemplate = "```question\n# Your question here\n```\n\n";
            break;
        }
        
        $scope.markdownContent = text.substring(0, start) + blockTemplate + text.substring(end);
        
        // Set cursor position after insertion
        $timeout(function() {
          textarea.focus();
          const newCursorPos = start + blockTemplate.indexOf('#');
          textarea.setSelectionRange(newCursorPos, newCursorPos + blockTemplate.split('\n')[1].length);
        });
        
        $scope.updatePreview();
      };

      // Validate the lesson content
      $scope.validateContent = function(showSuccess = true) {
        // Parse the markdown content
        parseMarkdown($scope.markdownContent).then(function(lesson) {
          // Send the lesson to the validation endpoint
          return $http.post('/api/lessons/validate', lesson);
        }).then(function(response) {
          $scope.validationErrors = [];
          if (showSuccess) {
            alert('Lesson content is valid!');
          }
        }).catch(function(error) {
          if (error.data && error.data.details) {
            $scope.validationErrors = [error.data.details];
          } else if (error.data && error.data.message) {
            $scope.validationErrors = [error.data.message];
          } else {
            $scope.validationErrors = ['An unknown error occurred during validation.'];
          }
        });
      };

      // Load a template
      $scope.loadTemplate = function() {
        $mdDialog.show({
          controller: 'LessonTemplateController',
          templateUrl: 'lesson-templates.html',
          parent: angular.element(document.body),
          clickOutsideToClose: true
        }).then(function(template) {
          if (template) {
            // Confirm before overwriting existing content
            if ($scope.markdownContent.trim() !== '') {
              if (!confirm('This will replace your current content. Continue?')) {
                return;
              }
            }
            
            $scope.markdownContent = template.content;
            $scope.updatePreview();
          }
        });
      };

      // Save the lesson as a draft
      $scope.saveAsDraft = function() {
        saveLesson(true);
      };

      // Save and publish the lesson
      $scope.saveAndPublish = function() {
        saveLesson(false);
      };

      // Save the lesson
      function saveLesson(isDraft) {
        // Parse the markdown content
        parseMarkdown($scope.markdownContent).then(function(lesson) {
          lesson.isDraft = isDraft;
          
          // Update or create the lesson
          let request;
          if ($scope.isEditing) {
            request = $http.put('/api/lessons/' + $scope.editingLessonId, lesson);
          } else {
            request = $http.post('/api/lessons', lesson);
          }
          
          return request;
        }).then(function(response) {
          alert(isDraft ? 'Lesson saved as draft!' : 'Lesson published successfully!');
          $mdDialog.hide(response.data);
        }).catch(function(error) {
          if (error.data && error.data.details) {
            alert('Error: ' + error.data.details);
          } else if (error.data && error.data.message) {
            alert('Error: ' + error.data.message);
          } else {
            alert('An unknown error occurred while saving the lesson.');
          }
        });
      }

      // Cancel and close the dialog
      $scope.cancel = function() {
        if ($scope.markdownContent.trim() !== '') {
          if (!confirm('You have unsaved changes. Are you sure you want to cancel?')) {
            return;
          }
        }
        $mdDialog.cancel();
      };
    }
  ]);

  // Register the lesson template controller
  angular.module('DockerPlay').controller('LessonTemplateController', [
    '$scope', '$mdDialog', '$http',
    function($scope, $mdDialog, $http) {
      $scope.templates = [
        {
          name: 'Basic Linux Commands',
          description: 'A simple lesson introducing basic Linux commands',
          content: `# Introduction to Linux Commands

This lesson introduces basic Linux commands for navigating the file system and working with files.

## Step 1: Navigating the File System

Let's start by exploring the file system. The \`pwd\` command shows your current directory:

\`\`\`docker
pwd
\`\`\`

\`\`\`expect
/home/user
\`\`\`

Now let's list the files in the current directory:

\`\`\`docker
ls -la
\`\`\`

## Step 2: Creating Files

Let's create a new file:

\`\`\`docker
echo "Hello, World!" > hello.txt
cat hello.txt
\`\`\`

\`\`\`expect
Hello, World!
\`\`\`

\`\`\`question
What command would you use to create a directory?
\`\`\`
`
        },
        {
          name: 'Docker Basics',
          description: 'Introduction to Docker commands and concepts',
          content: `# Docker Basics

This lesson introduces basic Docker commands and concepts.

## Step 1: Checking Docker Version

Let's start by checking the Docker version:

\`\`\`docker
docker --version
\`\`\`

## Step 2: Running a Container

Now let's run a simple container:

\`\`\`docker
docker run --rm hello-world
\`\`\`

## Step 3: Listing Containers

Let's list all running containers:

\`\`\`docker
docker ps
\`\`\`

\`\`\`question
What command would you use to stop a running container?
\`\`\`
`
        },
        {
          name: 'Git Basics',
          description: 'Introduction to Git version control',
          content: `# Git Basics

This lesson introduces basic Git commands for version control.

## Step 1: Initializing a Repository

Let's start by creating a new directory and initializing a Git repository:

\`\`\`docker
mkdir git-demo
cd git-demo
git init
\`\`\`

## Step 2: Making Changes

Now let's create a file and make our first commit:

\`\`\`docker
echo "# Git Demo" > README.md
git add README.md
git config --global user.email "user@example.com"
git config --global user.name "Example User"
git commit -m "Initial commit"
\`\`\`

## Step 3: Viewing History

Let's check the commit history:

\`\`\`docker
git log --oneline
\`\`\`

\`\`\`question
What command would you use to create a new branch in Git?
\`\`\`
`
        }
      ];

      $scope.selectTemplate = function(template) {
        $mdDialog.hide(template);
      };

      $scope.cancel = function() {
        $mdDialog.cancel();
      };
    }
  ]);

  // Extend the LessonService to include parsing and validation
  angular.module('DockerPlay').service('LessonService', ['$http', function($http) {
    // Keep existing methods
    var existingService = this;
    
    // Add new methods
    this.parseMarkdown = function(markdown) {
      return $http.post('/api/lessons/parse', { markdown: markdown }).then(function(response) {
        return response.data;
      });
    };
    
    this.validateLesson = function(lesson) {
      return $http.post('/api/lessons/validate', lesson);
    };
    
    return this;
  }]);

})();