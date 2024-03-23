SET NAMES utf8;
SET time_zone = '+00:00';
SET foreign_key_checks = 0;
SET sql_mode = 'NO_AUTO_VALUE_ON_ZERO';

SET NAMES utf8mb4;

USE `data`;

-- DROP TABLE IF EXISTS `posts`;
CREATE TABLE IF NOT EXISTS `posts` (
    `id` int NOT NULL AUTO_INCREMENT,
    `author` int,
    `date` datetime NOT NULL ON UPDATE CURRENT_TIMESTAMP,
    `title` text NOT NULL,
    `content` text NOT NULL,
    PRIMARY KEY (`id`),
    FOREIGN KEY (`author`) REFERENCES `users`(`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf16;


CREATE TABLE IF NOT EXISTS `users` (
    `id` int NOT NULL AUTO_INCREMENT,
    `username` text NOT NULL,
    `password` char(64) NOT NULL, -- hashed
    `salt` binary(16) NOT NULL,
    `access` int unsigned DEFAULT 0 NOT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO `users` (`id`, `name`, `password`, `salt`, `access`) 
VALUES (0, "admin", "FXIioaFEPRg588vm8xdLcWsg2L6tIrfMnz4ODcJs5ISO73Xp2yGfpDVrhRsb21BP", 0x8eef75e9db219fa4356b851b1bdb504f, 0b1111);



-- INSERT INTO `posts` (`id`, `date`, `title`, `content`) 
-- VALUES
--     (1, '2024-03-03 10:30:04', 'this is just a test post', 'hello world!\n\ntest'),
--     (2, '2024-03-03 10:31:55', 'hello',                    'world!');

