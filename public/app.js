var app = angular.module('dontFearTheSweeper', ['ngMask']);

app.controller('signup', function ($scope, $http, $window, $timeout) {

    $scope.weekdayMap = {
        0: 'Sunday',
        1: 'Monday',
        2: 'Tuesday',
        3: 'Wednesday',
        4: 'Thursday',
        5: 'Friday',
        6: 'Saturday'
    };

    $scope.nthWeekMap = {
        1: 'First',
        2: 'Second',
        3: 'Third',
        4: 'Fourth'
    };

    $scope.Alert = {
        timezone: "",
        times: [
            {
                weekday: "Weekday",
                nthWeek: "Nth"
            }
        ],
        phoneNumber: "",
        token: ""
    };

    $scope.setTimeZone = function(zone, buttonValue) {
        $scope.TimezoneButton = buttonValue;
        $scope.Alert.timezone = zone;
    };

    $scope.setNthWeek = function(n, index) {
        $scope.Alert.times[index].nthWeek = n;
    };

    $scope.setWeekday = function(day, index) {
        $scope.Alert.times[index].weekday = day;
    };

    $scope.addAlertTime = function() {
        $scope.Alert.times.push({weekday: "Weekday", nthWeek:"Nth"})
    };

    $scope.removeAlertTime = function(index) {
        $scope.Alert.times.splice(index, 1);
    };

    $scope.asYouType = function(number) {
        return new libphonenumber.asYouType('US').input(number)
    };

    $scope.isValidPhoneNumber = function(number) {
        return libphonenumber.isValidNumber(number, 'US');
    };

    var validateTimes = function() {
        for (var i = 0; i < $scope.Alert.times.length; i++) {
            var time = $scope.Alert.times[i];
            if (time.weekday === "Weekday" || time.nthWeek === "Nth") {
                return false
            }
        }
        return true
    };

    /**
     * Initialize Phone Verification
     */
    $scope.verificationCodeRequested = false;
    $scope.verificationCodeRequestError = false;

    $scope.startVerification = function () {

        if ($scope.Alert.timezone === "") {
            alert("must select timezone");
            return
        }

        var success = validateTimes();
        if (!success) {
            alert("must select a week and day");
            return
        }

        success = $scope.isValidPhoneNumber($scope.Alert.phoneNumber);
        if (!success) {
            alert("must have a valid phone number");
            return
        }

        $scope.verificationCodeRequested = false;
        $scope.verificationCodeRequestError = false;
        $http.post('/verification/start', $scope.Alert)
            .success(function (data, status, headers, config) {
                $scope.verificationCodeRequested = true;
            })
            .error(function (data, status, headers, config) {
                $scope.verificationCodeRequestError = true;
            });
    };

    /**
     * Verify phone token
     */
    $scope.verified = false;
    $scope.verifyError = false;
    $scope.verifyToken = function () {
        $http.post('/verification/verify', $scope.Alert)
            .success(function (data, status, headers, config) {
                $scope.verified = true;
            })
            .error(function (data, status, headers, config) {
                $scope.verifyError = true;
                // todo: should give meaningful feedback to user here
            });
    };

    /**
     * Delete all alerts of a given phone number
     */
    $scope.deleteAccount = function () {
        $http.post('/alerts/stop', $scope.Alert)
            .success(function (data, status, headers, config) {
                // todo: should give meaningful feedback to user here
                console.log("Delete started: ", data);
            })
            .error(function (data, status, headers, config) {
                // todo: should give meaningful feedback to user here
                console.error("remove alerts error: ", data);
            });
    };
});

app.directive('customValidation', function(){
    var previousInputValue = "1";
    return {
        require: 'ngModel',
        link: function(scope, element, attrs, modelCtrl) {
            modelCtrl.$parsers.push(function (inputValue) {
                console.log("inputValue: ", inputValue, "previousInputValue", previousInputValue, inputValue.length < previousInputValue.length);
                if (inputValue.length < previousInputValue.length) {
                    previousInputValue = inputValue;
                    return inputValue
                }
                var transformedInput = new libphonenumber.asYouType('US').input(inputValue);
                if (transformedInput !== inputValue) {
                    modelCtrl.$setViewValue(transformedInput);
                    modelCtrl.$render();
                }
                previousInputValue = transformedInput;
                return transformedInput;
            });
        }
    };
});
