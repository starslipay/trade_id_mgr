
CREATE TABLE `t_id_segment` (
  `scene_id` INTEGER NOT NULL COMMENT '场景标识',
  `max_allocated_id` BIGINT NOT NULL COMMENT '当前已分配出去的最大ID',
  `min_id` BIGINT NOT NULL COMMENT '最小ID用于隔离不同业务池的ID范围',
  `max_id` BIGINT NOT NULL COMMENT '最大ID用于隔离不同业务池的ID范围',
  `step_size` INTEGER NOT NULL COMMENT '每次批量领取号段步长',
  `memo` VARCHAR(128) NOT NULL COMMENT '备注',
  `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`scene_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT '全局唯一ID号段表';
-- goctl model mysql ddl -src id_generator.sql -dir .
