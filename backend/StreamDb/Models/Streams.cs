using System.ComponentModel.DataAnnotations;
using System.ComponentModel.DataAnnotations.Schema;

namespace StreamDb.Models;

public class Streams : BaseEntity 
{
    [Required]
    [MaxLength(100)]
    public string Title { get; set; } = null!;
    
    [Required]
    [MaxLength(100)]
    public string Description { get; set; } = null!;
    
    [Required]
    public DateTime StartTime { get; set; }
    
    [Required]
    public DateTime EndTime { get; set; }

    [Required]
    [MaxLength(100)]
    public string StreamKey { get; set; } = null!;
    
    [Required]
    [MaxLength(100)]
    public string Resolution { get; set; } = null!;
    
    [Required]
    public int Bitrate { get; set; } = 0;
    
    [Required]
    public int Framerate { get; set; } = 0;
    
    [Required]
    [MaxLength(100)]
    public string Codec { get; set; } = null!;

    [Required] public int ViewCount { get; set; } = 0;

    [Required]
    [MaxLength(100)]
    public string Protocol { get; set; } = null!;

    [Required] public EStreamStatus Status { get; set; } = EStreamStatus.SCHEDULED;
    
    [Column("user_id")]
    [Required]
    public int UserId { get; init; }
    
    public User User { get; init; }
    public ICollection<Comments> Comments { get; init; }
}
