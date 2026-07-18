CREATE DATABASE IF NOT EXISTS `id_generator_db` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

-- Use database
USE `id_generator_db`;
-- `id_generator_db`.`t_id_segment`
DROP TABLE IF EXISTS `t_id_segment`;
CREATE TABLE `t_id_segment` (
  `scene_id` BIGINT NOT NULL COMMENT '场景标识',
  `max_allocated_id` BIGINT NOT NULL COMMENT '当前已分配出去的最大ID',
  `min_id` BIGINT NOT NULL COMMENT '最小ID用于隔离不同业务池的ID范围',
  `max_id` BIGINT NOT NULL COMMENT '最大ID用于隔离不同业务池的ID范围',
  `step_size` BIGINT NOT NULL COMMENT '每次批量领取号段步长',
  `memo` VARCHAR(128) NOT NULL COMMENT '备注',
  `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`scene_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT '全局唯一ID号段表';

-- int64max = 9223372036854775807 ≈ 9200000000000000000
-- 初始化你的账单业务池
INSERT INTO t_id_segment(scene_id,max_allocated_id,min_id,max_id,step_size,memo)
VALUES (1,0,0,9200000000000000000,5000,'C2C');

INSERT INTO t_id_segment(scene_id,max_allocated_id,min_id,max_id,step_size,memo)
VALUES (0,10000000000,10000000000,9200000000000000000,100,'UID');

-- linux:  mysql -h 127.0.0.1 -P 3306 -u root -proot123456 < id_generator_init.sql
-- windows: Get-Content -Encoding UTF8 id_generator_init.sql | mysql -h 127.0.0.1 -P 3306 -u root -proot123456
