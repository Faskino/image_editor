CREATE USER 'mysql' @'%' IDENTIFIED BY 'Or4ndzov4Ceresn456';
GRANT ALL PRIVILEGES ON img_editor.* TO 'mysql' @'%';
FLUSH PRIVILEGES;
USE img_editor;
CREATE TABLE IF NOT EXISTS `users` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `username` varchar(255) NOT NULL,
  `password` varchar(255) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `username` (`username`)
);

CREATE TABLE IF NOT EXISTS `images` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `filename` varchar(255) NOT NULL,
  `filepath` varchar(255) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `user_id` int(11) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`),
  CONSTRAINT `images_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS `image_filters` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `image_id` int(11) NOT NULL,
  `contrast` int(11) DEFAULT '0',
  `vibrance` int(11) DEFAULT '0',
  `sepia` int(11) DEFAULT '0',
  `vignette` int(11) DEFAULT '0',
  `brightness` int(11) DEFAULT '0',
  `saturation` int(11) DEFAULT '0',
  `exposure` int(11) DEFAULT '0',
  `noise` int(11) DEFAULT '0',
  `sharpen` int(11) DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `image_id` (`image_id`),
  CONSTRAINT `image_filters_ibfk_1` FOREIGN KEY (`image_id`) REFERENCES `images` (`id`) ON DELETE CASCADE
);
