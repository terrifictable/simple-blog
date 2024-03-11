SET NAMES utf8;
SET time_zone = '+00:00';
SET foreign_key_checks = 0;
SET sql_mode = 'NO_AUTO_VALUE_ON_ZERO';

SET NAMES utf8mb4;

USE `data`;

-- DROP TABLE IF EXISTS `posts`;
CREATE TABLE IF NOT EXISTS `posts` (
    `id` int NOT NULL AUTO_INCREMENT,
    `date` datetime NOT NULL ON UPDATE CURRENT_TIMESTAMP,
    `title` text NOT NULL,
    `content` text NOT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf16;

INSERT INTO `posts` (`id`, `date`, `title`, `content`) 
VALUES
    (1, '2024-03-03 10:30:04', 'this is just a test post', 'hello world!\n\ntest'),
    (2, '2024-03-03 10:31:55', 'hello',                    'world!');
