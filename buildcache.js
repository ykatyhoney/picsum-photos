var sharp = require('sharp')
var path = require('path')
var async = require('async')
var config = require('./config')()
var fs = require('fs')

var imageProcessor = require('./imageProcessor')(sharp, path, config, fs)
var images = require(config.image_store_path)

fs.mkdir(config.cache_folder_path, function (error) {})

var index = process.argv[2] || 0
console.log('Start: %s', index)

if (index > 0) {
  images.splice(0, index)
}

async.eachLimit(images, 5, function (image, next) {
  var width = 458
  var height = 354
  var blur = false
  imageProcessor.getProcessedImage(width, height, null, false, false, image.filename, false, function (error, imagePath) {
    if (error) {
      console.log('filePath: ' + image.filename)
      console.log('imagePath: ' + imagePath)
      console.log('error: ' + err)
    }
    console.log('%s done', image.id)
    next()
  })
}, function (error) {
  console.log('Done')
})