'use strict';

angular.module('postyApp')
  .controller('MainCtrl', function ($scope,$http,$timeout) {
    $scope.showListMsg= function(message) {
        $scope.listMsg = message;
        $('#listMsg').fadeIn(600).delay(3000).fadeOut(600);
    };
    $scope.showPostMsg= function(message) {
        $scope.postMsg = message;
        $('#postMsg').fadeIn(600).delay(3000).fadeOut(600);
    };
    $scope.showListErrorMsg=function(message) {
      $scope.msgListError = message;
      $('#errorListMsg').fadeIn(600).delay(3000).fadeOut(600);
    };
    $scope.showPostErrorMsg=function(message) {
      $scope.errorPostMsg = message;
      $('#errorPostMsg').fadeIn(600).delay(3000).fadeOut(600);
    };
    $scope.loadPosts = function() {
        $http.get('/api/posts').success(function(data) {
            $scope.posts = data['data']
            $scope.showListMsg('Posts loaded!');
        }).error(function(data,status,headers,config) {
            console.log("Status", status);
            $scope.showListErrorMsg("Could not fetch posts :( but i'm not giving up");
        });
    };
    $scope.createPost= function(msg) {
      var postdata = {
        'data': {
          'message': msg,
        },
      };
      $http.post('/api/posts',postdata).success(function(data) {
        $scope.msg = "";
        $scope.posts.unshift(data.data);
        $timeout($scope.loadPosts, 5000);
        $scope.showPostMsg();
      }).error(function(data,status) {
        var title = 'Could not send your message :(';
        if (status === 400 && data.errors) {
          title = data.errors[0].title;
        }
        $scope.showPostErrorMsg(title);
      });

    };
    $scope.removePost= function(post) {
      $http.delete('/api/posts/'+post.id).success(function(data) {
        var index = $scope.posts.indexOf(post);
        $scope.posts.splice(index, 1);
        $scope.showListMsg('Message removed!');
        $scope.loadPosts();
      }).error(function(data,status) {
        var title = 'Could not remove this message :(';
        if (status === 400 && data.errors) {
          title = data.errors[0].title;
        }
        $scope.showListErrorMsg(title);
      });

    };
    $scope.loadPosts();
  });
