using System.ComponentModel.DataAnnotations;
using System.ComponentModel.DataAnnotations.Schema;

namespace StreamDb.Models;

public class Comments : BaseEntity
{
    [Column("message")]
    [Required]
    [MaxLength(255)]
    public  string Message { get; set; } = null!;
    
    [Column("user_id")]
    [Required]
    public int UserId { get; init; }
    
    [Column("stream_id")]
    [Required]
    public int StreamId { get; init; }
    
    public User User { get; init; }
    public Streams Stream { get; init; }
}