var app = angular.module("app", []);

app.controller("IndexController", function($scope, $http)
	{
		$scope.to = "";
		$scope.from = "";
		$scope.message = "";
		
		$scope.messageType = "basic";
		
		$scope.sticker = {
			id: "",
			imagePath: "",
			imageUrl: ""
		};
		
		$scope.messageType = "";
		$scope.messageTypes = [];
		
		$scope.fontSize = "";
		$scope.fontSizes = [];
		
		$scope.textColor = "";
		$scope.textColors = [];

		$scope.success = {
			show: false,
			imageUrl: ""
		};

		$scope.error = {
			show: false,
			message: ""
		};

		$scope.loading = {
			show: false
		};

		$scope.submit = {
			disabled: false
		};
		
		$scope.reset = {
			disabled: false
		};

		$scope.getMessageTypes = function()
		{
			$http.post("/MessageTypes").then(
				function(result)
				{
					$scope.messageTypes = result.data;
					
					console.log(result);
				},
				function(result)
				{
					console.log(result);
					$scope.handleHttpError("Error retrieving message types");
				}
			);
		}
		$scope.getMessageTypes();

		$scope.getFontSizes = function()
		{
			$http.post("/FontSizes").then(
				function(result)
				{
					$scope.fontSizes = result.data;
					
					console.log(result);
				},
				function(result)
				{
					console.log(result);
					$scope.handleHttpError("Error retrieving font sizes");
				}
			);
		}
		$scope.getFontSizes();
		
		$scope.getTextColors = function()
		{
			$http.post("/TextColors").then(
				function(result)
				{
					$scope.textColors = result.data;
					
					console.log(result);
				},
				function(result)
				{
					console.log(result);
					$scope.handleHttpError("Error retrieving text colors");
				}
			);
		}
		$scope.getTextColors();

		$scope.onCreateClick = function()
		{
			$scope.error.show = false;
			$scope.success.show = false;
			$scope.loading.show = true;
			
			$scope.submit.disabled = true;
			$scope.reset.disabled = true;

			var id = "";
			
			if ($scope.sticker != null){
				id = $scope.sticker.id;				
			}
			var data = {
				id: id,
				to: $scope.to,
				from: $scope.from,
				message: $scope.message,
				messageType: $scope.messageType ? $scope.messageType : "Basic",
				fontSize: $scope.fontSize ? $scope.fontSize : "70",
				textColor: $scope.textColor ? $scope.textColor : "Black"
			};
			
			$scope.sticker.id = "";
			$scope.sticker.imageSrc = "";
			$scope.sticker.imageViewUrl = "";

			$http.post("/create/", data).then(function(result)
			{
				var response = result.data;
				
				$scope.sticker.id = response.id;
				$scope.sticker.imageSrc = response.imageSrc;
				$scope.sticker.imageViewUrl = response.imageViewUrl;
				
				$scope.success.show = true;
				$scope.loading.show = false;
				
				$scope.submit.disabled = false;
				$scope.reset.disabled = false;
				
			}, function(result)
			{
				$scope.error.message = result;
				
				$scope.error.show = true;
				$scope.loading.show = false;
				
				$scope.submit.disabled = false;
				$scope.reset.disabled = false;
			});
		}
		
		$scope.onResetClick = function()
		{
			$scope.error.show = false;
			$scope.success.show = false;
			
			$scope.submit.disabled = true;
			$scope.reset.disabled = true;

			var id = "";
			
			if ($scope.sticker != null){
				id = $scope.sticker.id;				
			}
			var data = {
				id: id
			};
			
			$scope.sticker.id = "";
			$scope.sticker.imageSrc = "";
			$scope.sticker.imageViewUrl = "";

			$http.post("/delete/", data).then(function(result)
			{
				var response = result.data;
				$scope.success.show = true;
			}, function(result)
			{
				$scope.error.message = result;
				$scope.error.show = true;
			});

			$scope.submit.disabled = false;
		}
	}
);
