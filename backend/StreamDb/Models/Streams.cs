using System.ComponentModel.DataAnnotations;
using System.ComponentModel.DataAnnotations.Schema;

namespace StreamDb.Models;

public class Streams : BaseEntity 
{
    [Column("title")]
    [Required]
    public string Title { get; set; }
    
    [Column("description")]
    [Required]
    public string Description { get; set; }
    
    [Column("start_time")]
    [Required]
    public string StartTime { get; set; }
    
    [Column("end_time")]
    [Required]
    public string EndTime { get; set; }
    
    [Column("stream_key")]
    [Required]
    public string StreamKey { get; set; }
    
    [Column("resolution")]
    [Required]
    public string Resolution { get; set; }
    
    [Column("bitrate")]
    [Required]
    public string Bitrate { get; set; }
    
    [Column("framerate")]
    [Required]
    public string Framerate { get; set; }
    
    [Column("codec")]
    [Required]
    public string Codec { get; set; }
    
    [Column("view_count")]
    [Required]
    public int ViewCount { get; set; }
    
    [Column("protocol")]
    [Required]
    public int Protocol { get; set; }
    
    [Column("status")]
    [Required]
    public EStreamStatus Status { get; set; }
    
    [Column("user_id")]
    [Required]
    public int UserId { get; set; }
    
    public User User { get; set; }
    public ICollection<Comments> Comments { get; set; }
}
