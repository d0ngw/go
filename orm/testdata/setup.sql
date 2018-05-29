CREATE TABLE IF NOT EXISTS `tt` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `name` varchar(64)  DEFAULT NULL,
    `create_time` bigint(20) DEFAULT NULL,
    `f64` double DEFAULT NULL,
    PRIMARY KEY (`id`)) 
    ENGINE=InnoDB DEFAULT CHARSET=utf8;
